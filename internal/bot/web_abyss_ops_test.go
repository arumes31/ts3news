package bot

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"testing/quick"
	"time"
)

func TestAbyssFeatureRolloutIsStableAndBounded(t *testing.T) {
	for _, percent := range []int{0, 1, 50, 99, 100} {
		cfg := abyssFeatureConfig{liveActions: true, social: true, rollout: percent}
		for i := range 500 {
			uid := "player-" + strconv.Itoa(i)
			first := cfg.enabled("live_actions", uid)
			if first != cfg.enabled("live_actions", uid) {
				t.Fatalf("rollout assignment changed for %q", uid)
			}
			if percent == 0 && first || percent == 100 && !first {
				t.Fatalf("rollout %d produced invalid assignment for %q", percent, uid)
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
	server := &WebServer{abyssFeatures: abyssFeatureConfig{opsToken: "operator-secret"}}
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
