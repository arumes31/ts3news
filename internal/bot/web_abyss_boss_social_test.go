package bot

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestAbyssBossFinaleKeepsOneFrameForEachFinalRound(t *testing.T) {
	t.Parallel()
	frames := []combatTimelineFrame{
		{Round: 1, EnemyHP: 100}, {Round: 2, EnemyHP: 80},
		{Round: 2, EnemyHP: 70}, {Round: 3, EnemyHP: 40}, {Round: 4},
	}
	finale := abyssBossFinale(frames, 3)
	if len(finale) != 3 || finale[0].Round != 2 || finale[0].EnemyHP != 70 || finale[1].Round != 3 || finale[2].Round != 4 {
		t.Fatalf("unexpected boss finale: %#v", finale)
	}
}

func TestAbyssBossTauntsCrossEachThresholdOnce(t *testing.T) {
	t.Parallel()
	lines := abyssBossTaunts("The Watcher", []combatTimelineFrame{
		{Round: 1, EnemyHP: 60, EnemyMax: 100},
		{Round: 2, EnemyHP: 49, EnemyMax: 100},
		{Round: 3, EnemyHP: 20, EnemyMax: 100},
	})
	if len(lines) != 2 || !strings.Contains(lines[0], "R2") || !strings.Contains(lines[1], "R3") {
		t.Fatalf("unexpected boss taunts: %#v", lines)
	}
}

func TestAbyssBossIntroMetadataCoversNamedBosses(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		"Gorgoroth the Firelord", "Malakor the Voidweaver",
		"Azazoth the Slumbering Eye", "Abyssus, Heart of the Void",
	} {
		if title, tip := abyssBossTitle(name), abyssBossTip(name); title == "" || tip == "" {
			t.Errorf("intro metadata for %q = %q / %q", name, title, tip)
		}
	}
	if got := abyssBossTitle("Unknown Boss"); got != "Warden of the Deep" {
		t.Fatalf("fallback title = %q", got)
	}
}

func TestAbyssDeathKillerUsesStrongestSurvivor(t *testing.T) {
	t.Parallel()
	name, family := abyssDeathKiller([]*content.Mob{
		{Name: "Dead Giant", Type: content.MobType("Giant"), CurrentHP: 0, Stats: content.Stats{HP: 10_000}},
		{Name: "Living Rat", Type: content.MobType("Beast"), CurrentHP: 12},
		{Name: "Living Lich", Type: content.MobType("Undead"), CurrentHP: 800},
	})
	if name != "Living Lich" || family != "Undead" {
		t.Fatalf("death killer = %q, %q", name, family)
	}
	name, family = abyssDeathKiller(nil)
	if name != "The Abyss" || family != "Unknown" {
		t.Fatalf("fallback death killer = %q, %q", name, family)
	}
}

func TestAbyssPetMoodAndDuoBonus(t *testing.T) {
	t.Parallel()
	if mood, _, pct := abyssPetMood(20, 100, 100); mood != "scared" || pct != -2 {
		t.Fatalf("low-health mood = %q, %d", mood, pct)
	}
	if mood, _, pct := abyssPetMood(100, 100, 49); mood != "hungry" || pct != -2 {
		t.Fatalf("low-loyalty mood = %q, %d", mood, pct)
	}
	if got := abyssPetMoodScale(100, 2); got != 102 {
		t.Fatalf("positive mood scaling = %d", got)
	}
	users := []UserInCombat{
		{Stats: content.Stats{HP: 100, STR: 50, DEF: 25}, CurrentHP: 100},
		{Stats: content.Stats{HP: 200, STR: 100, DEF: 50}, CurrentHP: 200},
	}
	if applyAbyssDuoBonus(users, abyssDuoUnlocksAt-1) {
		t.Fatal("duo bonus unlocked early")
	}
	if !applyAbyssDuoBonus(users, abyssDuoUnlocksAt) || users[0].Stats.HP != 102 || users[1].Stats.STR != 102 {
		t.Fatalf("duo bonus not applied: %#v", users)
	}
}

func TestAbyssSocialPureRules(t *testing.T) {
	t.Parallel()
	if low, high, ok := abyssPair("zeta", "alpha"); !ok || low != "alpha" || high != "zeta" {
		t.Fatalf("canonical pair = %q, %q, %v", low, high, ok)
	}
	if _, _, ok := abyssPair("same", "same"); ok {
		t.Fatal("self pair accepted")
	}
	if stat, value := abyssPetLowestStat(50, 20, 30); stat != "def" || value != 20 {
		t.Fatalf("lowest pet stat = %q, %d", stat, value)
	}
	if got := abyssWeeklyBossDamage(0); got != 1 {
		t.Fatalf("minimum weekly boss damage = %d", got)
	}
	if got := abyssWeeklyBossDamage(123.9); got != 3_097 {
		t.Fatalf("weekly boss damage = %d", got)
	}
	if !validAbyssSpectatorSessionID(strings.Repeat("a1", 16)) {
		t.Fatal("valid spectator session rejected")
	}
	for _, invalid := range []string{"", strings.Repeat("a", 31), strings.Repeat("A", 32), strings.Repeat("g", 32)} {
		if validAbyssSpectatorSessionID(invalid) {
			t.Fatalf("invalid spectator session accepted: %q", invalid)
		}
	}
}

func TestAbyssWeeklyBossesHaveDistinctValidDropTables(t *testing.T) {
	t.Parallel()

	bosses := []string{
		"Nhal, the Starved Horizon",
		"Veyra of the Thousand Eyes",
		"The Iron Leviathan",
		"Mournroot Prime",
	}
	validMaterials := map[string]bool{"dust": true, "shard": true, "core": true, "prism": true}
	summaries := make(map[string]bool, len(bosses))
	for _, boss := range bosses {
		table := abyssWeeklyBossDropTable(boss)
		weight := 0
		for _, drop := range table {
			weight += drop.Weight
			if !validMaterials[drop.Material] || drop.Amount <= 0 || drop.Weight <= 0 {
				t.Errorf("%q has invalid drop: %#v", boss, drop)
			}
		}
		if weight != 100 {
			t.Errorf("%q drop weight = %d", boss, weight)
		}
		summary := abyssWeeklyBossDropSummary(boss)
		if summary == "" || summaries[summary] {
			t.Errorf("%q has empty or duplicate drop table %q", boss, summary)
		}
		summaries[summary] = true
	}
}

func TestAbyssWeeklyBossDropIsStableForPlayerDay(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	first := abyssWeeklyBossDropFor("Veyra of the Thousand Eyes", "2026-W35", "player-1", now)
	second := abyssWeeklyBossDropFor("Veyra of the Thousand Eyes", "2026-W35", "player-1", now.Add(8*time.Hour))
	if first != second {
		t.Fatalf("same-day drop changed: %#v != %#v", first, second)
	}
}

func TestAbyssSocialPersistenceAndUIContracts(t *testing.T) {
	t.Parallel()
	root := abyssAAARepositoryRoot(t)
	checks := map[string][]string{
		filepath.Join(root, "internal", "db", "migrations", "0081_abyss_social.up.sql"): {
			"active_slot", "autoskills", "abyss_pet_memorials", "abyss_deaths", "abyss_helper_bonds",
			"abyss_weekly_rivals", "abyss_weekly_boss_contributions", "PRIMARY KEY (week_key, client_uid, contribution_date)",
		},
		filepath.Join(root, "internal", "bot", "webassets", "abyss_social.html"): {
			"Companion command", "Weekly rival", "Revenge mark", "WEEKLY SERVER BOSS", "First-kill trophies",
			"Rotating drops:", "/api/abyss/social/pet/train", "/api/abyss/social/weekly_boss",
		},
		filepath.Join(root, "internal", "bot", "webassets", "abyss_spectate.html"): {
			"READ-ONLY LIVE FEED", "textContent", "replaceChildren", "/api/abyss/spectate",
		},
	}
	for path, required := range checks {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		source := string(raw)
		for _, token := range required {
			if !strings.Contains(source, token) {
				t.Errorf("%s is missing %q", filepath.Base(path), token)
			}
		}
	}
}

func TestAbyssPetTrainingCommitsGoldAndLowestStatAtomically(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	const uid = "pet-trainer"
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT str,def,spd,CASE WHEN trained_on=CURRENT_DATE").
		WithArgs(int64(7), uid).
		WillReturnRows(sqlmock.NewRows([]string{"str", "def", "spd", "training_count"}).AddRow(200, 100, 150, 1))
	mock.ExpectExec("UPDATE users SET gold=gold-\\$1").
		WithArgs(abyssPetTrainingCost, uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE user_pets SET def=def+$1,trained_on=CURRENT_DATE,training_count=$2 WHERE pet_id=$3 AND client_uid=$4")).
		WithArgs(1, 2, int64(7), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT gold FROM users WHERE client_uid=$1")).
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(9_000))

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/social/pet/train", strings.NewReader(`{"pet_id":7}`))
	response := httptest.NewRecorder()
	server.handleAbyssPetTrain(response, request, uid)
	if body := response.Body.String(); !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"stat":"DEF"`) {
		t.Fatalf("pet training response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssPetTrainingRollsBackWhenGoldIsInsufficient(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	const uid = "pet-trainer"
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT str,def,spd,CASE WHEN trained_on=CURRENT_DATE").
		WithArgs(int64(7), uid).
		WillReturnRows(sqlmock.NewRows([]string{"str", "def", "spd", "training_count"}).AddRow(200, 100, 150, 1))
	mock.ExpectExec("UPDATE users SET gold=gold-\\$1").
		WithArgs(abyssPetTrainingCost, uid).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/social/pet/train", strings.NewReader(`{"pet_id":7}`))
	response := httptest.NewRecorder()
	server.handleAbyssPetTrain(response, request, uid)
	if body := response.Body.String(); !strings.Contains(body, `"ok":false`) || !strings.Contains(body, "not enough gold") {
		t.Fatalf("pet training response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssSpectatorSnapshotRedactsPrivateCombatState(t *testing.T) {
	t.Parallel()
	snapshot := abyssLiveSnapshot{
		OwnerUID: "owner-secret", Options: []abyssLiveOption{{Kind: "item"}},
		Queued: &abyssLiveAction{Kind: "skill"}, Recommended: &abyssLiveRecommendation{},
		Policy: abyssLivePolicy{TimeoutAction: "best"}, CanConfigure: true,
		TimeBankMS: 5_000, RandomSeed: [2]uint64{1, 2}, RandomDraws: 9,
		Result:       map[string]any{"gold": 999},
		Allies:       []abyssLiveCombatantView{{ID: "ally:private-uid", Name: "Ada", IsSelf: true, Ready: true}},
		Enemies:      []abyssLiveCombatantView{{ID: "enemy:42", Name: "Watcher"}},
		Initiative:   []abyssLiveInitiativeEntry{{ID: "ally:private-uid"}},
		EnemyIntents: []abyssLiveEnemyIntent{{EnemyID: "enemy:42", TargetID: "ally:private-uid"}},
		Social: abyssLiveSocialSnapshot{
			PreferredRole: "support", Signals: []abyssLiveSocialSignal{{TargetID: "enemy:42"}},
			Members: []abyssLiveMemberPresence{{Name: "Ada"}},
		},
	}
	public := sanitizeAbyssSpectatorSnapshot(snapshot)
	if public.OwnerUID != "" || public.Options != nil || public.Queued != nil || public.Recommended != nil || public.Result != nil {
		t.Fatalf("private controls remained in spectator snapshot: %#v", public)
	}
	if public.RandomSeed != [2]uint64{} || public.RandomDraws != 0 || public.TimeBankMS != 0 {
		t.Fatal("private deterministic state remained in spectator snapshot")
	}
	if public.Allies[0].ID != "ally:1" || public.Enemies[0].ID != "enemy:1" || public.Initiative[0].ID != "ally:1" {
		t.Fatalf("combat IDs were not anonymized: %#v", public)
	}
	if public.EnemyIntents[0].EnemyID != "enemy:1" || public.EnemyIntents[0].TargetID != "ally:1" {
		t.Fatalf("intent IDs were not anonymized: %#v", public.EnemyIntents[0])
	}
	if public.Social.PreferredRole != "" || public.Social.Signals != nil || len(public.Social.Members) != 1 {
		t.Fatalf("social state was not reduced to public presence: %#v", public.Social)
	}
}
