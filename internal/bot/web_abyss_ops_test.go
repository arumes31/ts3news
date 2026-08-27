package bot

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"testing/quick"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssFeatureRolloutIsStableAndBounded(t *testing.T) {
	for _, percent := range []int{0, 1, 50, 99, 100} {
		cfg := &abyssFeatureConfig{
			liveActions: true, social: true, tree: true, forge: true,
			liveRollout: percent, socialRollout: percent, treeRollout: percent, forgeRollout: percent,
		}
		for _, feature := range []string{"live_actions", "social", "tree", "forge"} {
			for i := range 500 {
				uid := "player-" + strconv.Itoa(i)
				first := cfg.enabled(feature, uid)
				if first != cfg.enabled(feature, uid) {
					t.Fatalf("%s rollout assignment changed for %q", feature, uid)
				}
				if percent == 0 && first || percent == 100 && !first {
					t.Fatalf("%s rollout %d produced invalid assignment for %q", feature, percent, uid)
				}
			}
		}
	}
}

func TestAbyssBalanceGate(t *testing.T) {
	lastDifficulty := 0.0
	lastRisk := 0
	for depth := 1; depth <= 250; depth++ {
		difficulty, _ := abyssDifficulty(depth)
		if math.IsNaN(difficulty) || math.IsInf(difficulty, 0) || difficulty < lastDifficulty {
			t.Fatalf("difficulty is invalid or decreased at depth %d: %v -> %v", depth, lastDifficulty, difficulty)
		}
		risk := abyssRiskPct(depth, abyssTiers["normal"], 1000)
		if risk < lastRisk || risk < 0 || risk > 100 {
			t.Fatalf("risk is invalid or decreased at depth %d: %d -> %d", depth, lastRisk, risk)
		}
		if level := abyssMobLevel(depth, 100); level < 1 || level > 200 {
			t.Fatalf("mob level escaped its production cap at depth %d: %d", depth, level)
		}
		if reward := abyssFloorBonus(depth, 100); reward <= 0 || reward > int64(depth)*abyssFloorBonusMaxPer {
			t.Fatalf("reward escaped its production cap at depth %d: %d", depth, reward)
		}
		lastDifficulty, lastRisk = difficulty, risk
	}
	if early, deep := modeledAbyssWinRate(5, 1000), modeledAbyssWinRate(80, 1000); early < 0.55 || deep >= early {
		t.Fatalf("balance model does not preserve the push-your-luck curve: early=%.3f deep=%.3f", early, deep)
	}
}

func modeledAbyssWinRate(depth int, combatRating float64) float64 {
	risk := float64(abyssRiskPct(depth, abyssTiers["normal"], combatRating)) / 100
	return 1 / (1 + math.Exp(8*(risk-0.65)))
}

func TestAbyssTargetingProperties(t *testing.T) {
	property := func(a, b, c uint16) bool {
		hp := []int{int(a%1000) + 1, int(b%1000) + 1, int(c%1000) + 1}
		combat := abyssLiveCombat{
			enemies: []abyssLiveCombatantView{
				{ID: "enemy:0", HP: hp[0]}, {ID: "enemy:1", HP: hp[1]}, {ID: "enemy:2", HP: hp[2]},
			},
			policies: map[string]abyssLivePolicy{"user": {AttackPriority: "lowest_hp", SkillPriority: "highest_hp"}},
		}
		attack := combat.selectEnemyLocked("user", "attack")
		skill := combat.selectEnemyLocked("user", "skill")
		return attack.HP == min(hp[0], hp[1], hp[2]) && skill.HP == max(hp[0], hp[1], hp[2])
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 2000}); err != nil {
		t.Fatal(err)
	}
}

func FuzzAbyssMalformedActionRequest(f *testing.F) {
	for _, seed := range [][]byte{
		{}, []byte("{"), []byte("null"), []byte(`{"kind":"attack","target_id":"enemy:0"}`),
		bytes.Repeat([]byte("x"), 1024),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, body []byte) {
		req := httptest.NewRequest(http.MethodPost, "/api/abyss/combat/action", bytes.NewReader(body))
		var action abyssLiveAction
		_ = readJSON(req, &action)
	})
}

type lockedResponseRecorder struct {
	mu sync.Mutex
	*httptest.ResponseRecorder
}

func (r *lockedResponseRecorder) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.ResponseRecorder.Write(p)
}

func (r *lockedResponseRecorder) Flush() {}

func TestAbyssSSELoad(t *testing.T) {
	const clients = 64
	const eventsPerClient = 50
	var wg sync.WaitGroup
	for client := range clients {
		wg.Add(1)
		go func(client int) {
			defer wg.Done()
			recorder := &lockedResponseRecorder{ResponseRecorder: httptest.NewRecorder()}
			for event := range eventsPerClient {
				snapshot := abyssLiveSnapshot{OK: true, SessionID: strconv.Itoa(client), Version: int64(event)}
				if err := writeAbyssLiveEvent(recorder, recorder, snapshot); err != nil {
					t.Errorf("client %d event %d: %v", client, event, err)
					return
				}
			}
			var snapshot abyssLiveSnapshot
			line := bytes.Split(recorder.Body.Bytes(), []byte("\n"))
			if len(line) < 3 || json.Unmarshal(bytes.TrimPrefix(line[2], []byte("data: ")), &snapshot) != nil {
				t.Errorf("client %d produced malformed SSE", client)
			}
		}(client)
	}
	wg.Wait()
}

func TestAbyssOpsRequiresDedicatedToken(t *testing.T) {
	server := &WebServer{abyssFeatures: &abyssFeatureConfig{opsToken: "operator-secret"}}
	request := httptest.NewRequest(http.MethodGet, "/api/abyss/ops", nil)
	recorder := httptest.NewRecorder()
	server.handleAbyssOps(recorder, request, "player")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("ordinary player status = %d, want 404", recorder.Code)
	}
	request = httptest.NewRequest(http.MethodGet, "/api/abyss/ops", nil)
	request.Header.Set("Authorization", "Bearer operator-secret")
	recorder = httptest.NewRecorder()
	server.handleAbyssOps(recorder, request, "player")
	if recorder.Code != http.StatusOK {
		t.Fatalf("operator status = %d, want 200", recorder.Code)
	}
}

func TestAbyssRegistryHealthDetectsOrphansAndStaleSessions(t *testing.T) {
	server := &WebServer{}
	combat := &abyssLiveCombat{
		id: "stale", phase: "planning", deadline: time.Now().Add(-3 * time.Minute),
		participants: map[string]bool{"user": true},
	}
	server.liveCombats.Store(combat.id, combat)
	active, stale, orphan := server.abyssRegistryHealth(time.Now())
	if active != 1 || stale != 1 || orphan != 1 {
		t.Fatalf("health = active:%d stale:%d orphan:%d", active, stale, orphan)
	}
}

func TestAbyssContentReferencesAndAnomalyThresholds(t *testing.T) {
	if err := validateAbyssContentReferences(); err != nil {
		t.Fatal(err)
	}
	if abyssEconomyAnomaly(10, 100_000) || !abyssEconomyAnomaly(10, -1) || !abyssEconomyAnomaly(10, 30_000_000) {
		t.Fatal("economy anomaly thresholds do not separate ordinary and impossible rewards")
	}
}

func TestAbyssFeatureConfigUpdateAndRewardAssignment(t *testing.T) {
	cfg := newAbyssFeatureConfig(nil)
	enabled := true
	snapshot, experimentChanged, err := cfg.update(abyssFeatureUpdate{Feature: "reward_experiment", Enabled: &enabled})
	if err != nil || !experimentChanged || !snapshot.RewardExperimentEnabled {
		t.Fatalf("enable reward experiment = %#v, changed=%v, err=%v", snapshot, experimentChanged, err)
	}
	bonus := 750
	snapshot, experimentChanged, err = cfg.update(abyssFeatureUpdate{Feature: "reward_treatment_bonus", BonusBPS: &bonus})
	if err != nil || !experimentChanged || snapshot.RewardTreatmentBonusBPS != 750 {
		t.Fatalf("set reward bonus = %#v, changed=%v, err=%v", snapshot, experimentChanged, err)
	}
	if _, _, err := cfg.update(abyssFeatureUpdate{Feature: "reward_treatment_bonus", BonusBPS: ptr(2501)}); err == nil {
		t.Fatal("out-of-bounds treatment bonus was accepted")
	}

	assignment := cfg.rewardAssignment("stable-player")
	if assignment != cfg.rewardAssignment("stable-player") {
		t.Fatal("reward cohort assignment changed for the same player")
	}
	adjusted := applyAbyssRewardAssignment(10_000, assignment)
	if assignment.Cohort == "treatment" && adjusted != 10_750 {
		t.Fatalf("treatment reward = %d, want 10750", adjusted)
	}
	if assignment.Cohort != "treatment" && adjusted != 10_000 {
		t.Fatalf("non-treatment reward = %d, want 10000", adjusted)
	}
}

func TestAbyssFeatureConfigIsConcurrencySafe(t *testing.T) {
	cfg := newAbyssFeatureConfig(nil)
	var wait sync.WaitGroup
	for index := range 50 {
		wait.Go(func() {
			enabled := index%2 == 0
			_, _, _ = cfg.update(abyssFeatureUpdate{Feature: "social", Enabled: &enabled})
			_ = cfg.enabled("social", strconv.Itoa(index))
			_ = cfg.snapshot()
		})
	}
	wait.Wait()
}

func TestAbyssRewardExperimentGuardrails(t *testing.T) {
	features := abyssFeatureSnapshot{RewardExperimentEnabled: true, RewardExperimentRevision: 3}
	cfg := &abyssFeatureConfig{rewardExperiment: true, rewardRollout: 100, rewardBonusBPS: 500, experimentRev: 3}
	server := &WebServer{abyssFeatures: cfg}
	metrics := &server.abyssOps
	metrics.resetRewardExperiment(3)
	control := abyssRewardAssignment{Cohort: "control", MultiplierBPS: abyssRewardBaseBPS, Revision: 3}
	treatment := abyssRewardAssignment{Cohort: "treatment", MultiplierBPS: 10_500, Revision: 3}
	for index := range abyssExperimentGuardrailSample {
		metrics.observeRewardExperiment(control, 100, true, true, false)
		metrics.observeRewardExperiment(treatment, 105, true, index < 15, false)
	}
	snapshot := metrics.rewardExperimentSnapshot(features)
	if snapshot.Status != "halted" {
		t.Fatalf("guardrail status = %q, want halted", snapshot.Status)
	}
	if snapshot.Cohorts["treatment"].DeathRate != 0.25 {
		t.Fatalf("treatment death rate = %v, want 0.25", snapshot.Cohorts["treatment"].DeathRate)
	}
	for index := 0; index < 1_000; index++ {
		uid := "treatment-" + strconv.Itoa(index)
		if cfg.rewardAssignment(uid).Cohort != "treatment" {
			continue
		}
		assignment := server.abyssRewardAssignment(uid)
		if assignment.Cohort != "holdout" || assignment.MultiplierBPS != abyssRewardBaseBPS {
			t.Fatalf("guarded assignment = %#v, want zero-uplift holdout", assignment)
		}
		return
	}
	t.Fatal("test could not find a deterministic treatment assignment")
}

func TestAbyssBalanceSnapshotUsesAuthoritativeRunHistory(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	day := time.Date(2026, time.August, 24, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT DATE_TRUNC('day', created_at), COUNT(*)")).
		WillReturnRows(sqlmock.NewRows([]string{"day", "runs", "deaths", "loot", "floors"}).AddRow(day, int64(10), int64(2), int64(30), int64(20)))
	server := &WebServer{bot: &Bot{DB: database}}
	snapshot := server.abyssBalanceSnapshot(t.Context())
	if !snapshot.Available || len(snapshot.Days) != 1 {
		t.Fatalf("balance snapshot = %#v", snapshot)
	}
	if snapshot.Days[0].DeathRate != 0.2 || snapshot.Days[0].DropsPerFloor != 1.5 {
		t.Fatalf("balance point = %#v", snapshot.Days[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssOpsRuntimeUpdateRequiresTokenAndStrictJSON(t *testing.T) {
	server := &WebServer{abyssFeatures: &abyssFeatureConfig{opsToken: "operator-secret", liveRollout: 100}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/ops", bytes.NewBufferString(`{"feature":"social","enabled":false}`))
	recorder := httptest.NewRecorder()
	server.handleAbyssOps(recorder, request, "player")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unauthorized update status = %d, want 404", recorder.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/abyss/ops", bytes.NewBufferString(`{"feature":"social","enabled":false,"unexpected":1}`))
	request.Header.Set("Authorization", "Bearer operator-secret")
	recorder = httptest.NewRecorder()
	server.handleAbyssOps(recorder, request, "player")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status = %d, want 400", recorder.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/abyss/ops", bytes.NewBufferString(`{"feature":"social","enabled":false}`))
	request.Header.Set("Authorization", "Bearer operator-secret")
	recorder = httptest.NewRecorder()
	server.handleAbyssOps(recorder, request, "player")
	if recorder.Code != http.StatusOK || server.abyssFeatures.snapshot().Social {
		t.Fatalf("authorized update status=%d config=%#v", recorder.Code, server.abyssFeatures.snapshot())
	}
}

func ptr[T any](value T) *T { return &value }
