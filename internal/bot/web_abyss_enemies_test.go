package bot

import (
	"database/sql"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

type fixedCombatRandom struct {
	float float64
	intn  int
}

func (r fixedCombatRandom) Float64() float64 { return r.float }

func (r fixedCombatRandom) IntN(n int) int {
	if n <= 0 {
		panic("invalid random bound")
	}
	return r.intn % n
}

func TestAbyssEnemyIdentityIsRecognizable(t *testing.T) {
	tests := []struct {
		mob     content.Mob
		role    string
		faction string
	}{
		{content.Mob{Name: "Frost Lich", Spells: []content.Skill{{Name: "Bolt"}}, Stats: content.Stats{STR: 20, DEF: 5}}, "caster", "Graveborn"},
		{content.Mob{Name: "Armored Orc", Stats: content.Stats{STR: 5, DEF: 20}}, "guardian", "Deepclaw Clan"},
		{content.Mob{Name: "Treasure Goblin", Type: content.MobTreasureGoblin}, "runner", "Deepclaw Clan"},
		{content.Mob{Name: "Hazard: Volatile Rift"}, "hazard", "Abyssal Phenomena"},
	}
	for _, test := range tests {
		if got := abyssEnemyRole(&test.mob); got != test.role {
			t.Errorf("abyssEnemyRole(%q) = %q, want %q", test.mob.Name, got, test.role)
		}
		if got := abyssEnemyFaction(&test.mob); got != test.faction {
			t.Errorf("abyssEnemyFaction(%q) = %q, want %q", test.mob.Name, got, test.faction)
		}
		if abyssEnemyPattern(&test.mob) == "" {
			t.Errorf("abyssEnemyPattern(%q) is empty", test.mob.Name)
		}
	}
}

func TestPrepareAbyssEnemiesAddsMechanicalVariety(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New(): %v", err)
	}
	defer func() { _ = db.Close() }()
	mock.ExpectQuery("SELECT value FROM app_meta").
		WithArgs(abyssNemesisPrefix + "user").
		WillReturnError(sql.ErrNoRows)
	bot := &Bot{DB: db}
	mobs := []content.Mob{
		{Name: "Void Knight", Type: content.MobElite, Stats: content.Stats{HP: 200, STR: 30, DEF: 10}, MaxHP: 200},
		{Name: "Void Seer", Type: content.MobElite, Stats: content.Stats{HP: 180, STR: 25, DEF: 8}, MaxHP: 180},
	}

	prepared, logs := bot.prepareAbyssEnemies("user", 20, mobs, fixedCombatRandom{})
	if len(prepared) != 5 {
		t.Fatalf("prepared enemy count = %d, want elite pack + hazard + invader + named rare", len(prepared))
	}
	if prepared[0].MaxBreak <= 0 || prepared[0].Break != prepared[0].MaxBreak {
		t.Fatalf("elite break bar was not initialized: %+v", prepared[0])
	}
	if len(prepared[0].Effects) == 0 {
		t.Fatal("elite did not receive an affix")
	}
	joined := strings.Join(logs, "\n")
	for _, marker := range []string{"Volatile Rift", "invasion", "NAMED RARE", "Sableclaw's Phase Fang", "pack synergy", "Coordinated elite"} {
		if !strings.Contains(joined, marker) {
			t.Errorf("enemy-system logs missing %q:\n%s", marker, joined)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestAbyssNamedRaresHaveFixedDistinctSignatureDrops(t *testing.T) {
	t.Parallel()

	names := make(map[string]bool, len(abyssNamedRareDefinitions))
	signatures := make(map[string]bool, len(abyssNamedRareDefinitions))
	for _, definition := range abyssNamedRareDefinitions {
		if names[definition.Name] || signatures[definition.Signature] {
			t.Fatalf("duplicate named rare definition: %#v", definition)
		}
		names[definition.Name] = true
		signatures[definition.Signature] = true
		label, grant, ok := abyssNamedRareDrop(definition.Name)
		if !ok || grant.Type != "unique" || grant.UniqName != definition.Signature || grant.UniqPow != definition.Power {
			t.Errorf("signature drop for %q = %q, %#v, %v", definition.Name, label, grant, ok)
		}
	}
	if _, _, ok := abyssNamedRareDrop("ordinary mob"); ok {
		t.Fatal("ordinary mob received a named-rare signature drop")
	}
}

func TestAbyssNamedRareSpawnUsesDepthGateAndCatalog(t *testing.T) {
	t.Parallel()

	if _, _, ok := abyssNamedRareSpawn(5, fixedCombatRandom{}); ok {
		t.Fatal("named rare spawned before depth gate")
	}
	if _, _, ok := abyssNamedRareSpawn(20, fixedCombatRandom{float: 0.50}); ok {
		t.Fatal("named rare ignored spawn chance")
	}
	mob, signature, ok := abyssNamedRareSpawn(20, fixedCombatRandom{float: 0.03, intn: 2})
	if !ok || mob.Name != "Orryx, the Drowned Star" || signature != "Orryx's Tidal Lens" || mob.Type != content.MobElite {
		t.Fatalf("named rare spawn = %#v, %q, %v", mob, signature, ok)
	}
}

func TestAbyssNamedRareGrantRemainsRetryableOnDatabaseFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	storageErr := errors.New("storage unavailable")
	mock.ExpectExec("INSERT INTO user_unique_items").
		WithArgs("rare-hunter", "Orryx's Tidal Lens", content.RarityEpic, float64(10)).
		WillReturnError(storageErr)
	bot := &Bot{DB: db}
	_, grant, ok := abyssNamedRareDrop("Orryx, the Drowned Star")
	if !ok {
		t.Fatal("named rare drop missing")
	}
	if err := bot.applyAbyssLootGrant("rare-hunter", grant); !errors.Is(err, storageErr) {
		t.Fatalf("unique grant error = %v, want %v", err, storageErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssUniqueGrantReliesOnDatabaseCounterTrigger(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectExec("INSERT INTO user_unique_items").
		WithArgs("rare-hunter", "Sableclaw's Phase Fang", content.RarityEpic, float64(8)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	bot := &Bot{DB: db}
	if err := bot.grantAbyssUnique("rare-hunter", "Sableclaw's Phase Fang", content.RarityEpic, 8); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestScheduledAbyssEliteAuraPromotesCarrierEveryThirdFloor(t *testing.T) {
	t.Parallel()

	base := []content.Mob{
		{Name: "Scout", Type: content.MobCommon, Stats: content.Stats{HP: 100, STR: 20, DEF: 10, SPD: 5}},
		{Name: "Brute", Type: content.MobCommon, Stats: content.Stats{HP: 200, STR: 30, DEF: 20, SPD: 4}},
	}
	unchanged, log := applyScheduledAbyssEliteAura(5, append([]content.Mob(nil), base...))
	if log != "" || unchanged[1].Type != content.MobCommon {
		t.Fatalf("non-third floor received elite aura: %q, %#v", log, unchanged)
	}

	empowered, log := applyScheduledAbyssEliteAura(6, append([]content.Mob(nil), base...))
	if empowered[1].Type != content.MobElite || empowered[1].Stats.DEF != 22 {
		t.Fatalf("scheduled carrier = %#v", empowered[1])
	}
	if empowered[0].Stats.DEF != 11 || !strings.Contains(log, "Iron Canticle") || !strings.Contains(log, "+10% DEF") {
		t.Fatalf("scheduled aura did not empower pack: %#v, %q", empowered, log)
	}
	foundArmored := false
	for _, effect := range empowered[1].Effects {
		foundArmored = foundArmored || effect == content.EffectArmored
	}
	if !foundArmored {
		t.Fatal("elite aura carrier is missing its visible affix")
	}
}

func TestScheduledAbyssEliteAurasRotate(t *testing.T) {
	t.Parallel()

	want := map[int]string{3: "Blood Chorus", 6: "Iron Canticle", 9: "Gale Hymn", 12: "Blood Chorus"}
	for depth, name := range want {
		aura, ok := abyssEliteAuraForDepth(depth)
		if !ok || aura.Name != name {
			t.Errorf("depth %d aura = %#v, %v; want %q", depth, aura, ok, name)
		}
	}
}

func TestAbyssBreakAndBossAdaptation(t *testing.T) {
	mob := &content.Mob{
		Name:     "Boss",
		Type:     content.MobBoss,
		Stats:    content.Stats{HP: 500, SPD: 30},
		Break:    20,
		MaxBreak: 20,
	}
	logs := []string{}
	applyAbyssBreakDamage(mob, 100, &logs)
	if mob.Break != 0 || mob.Stats.SPD != 0 || len(logs) != 1 {
		t.Fatalf("break result = break %d speed %d logs %v", mob.Break, mob.Stats.SPD, logs)
	}

	combat := &abyssLiveCombat{actionCounts: map[string]int{"attack": 3}}
	adaptationLogs := combat.applyLiveBossAdaptation([]*content.Mob{mob}, nil)
	if combat.bossAdaptation != "armored" || len(adaptationLogs) != 1 {
		t.Fatalf("boss adaptation = %q logs %v", combat.bossAdaptation, adaptationLogs)
	}
	found := false
	for _, effect := range mob.Effects {
		found = found || effect == content.EffectArmored
	}
	if !found {
		t.Fatal("repeated attacks did not add the boss's Armored counter")
	}
}

func TestHandleAbyssBossPracticeRequiresUnlockedBossAndGrantsNoRewards(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New(): %v", err)
	}
	defer func() { _ = db.Close() }()
	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(depth\\),0\\)").
		WithArgs("user", "The Watcher").
		WillReturnRows(sqlmock.NewRows([]string{"depth"}).AddRow(25))
	server := &WebServer{bot: &Bot{DB: db}}
	request := httptest.NewRequest("POST", "/api/abyss/practice", strings.NewReader(`{"name":"The Watcher","tactic":"aggressive"}`))
	recorder := httptest.NewRecorder()

	server.handleAbyssBossPractice(recorder, request, "user")
	body := recorder.Body.String()
	if !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"rewards":false`) ||
		!strings.Contains(body, `"drill"`) || !strings.Contains(body, `"tactic":"aggressive"`) ||
		!strings.Contains(body, `"rounds":`) || !strings.Contains(body, `"player_max_hp":`) {
		t.Fatalf("practice response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestSimulateAbyssBossPracticeIsDeterministicAndTactical(t *testing.T) {
	t.Parallel()

	balanced := simulateAbyssBossPractice("Abyssus, Heart of the Void", 50, 250, "balanced")
	repeat := simulateAbyssBossPractice("Abyssus, Heart of the Void", 50, 250, "balanced")
	aggressive := simulateAbyssBossPractice("Abyssus, Heart of the Void", 50, 250, "aggressive")
	defensive := simulateAbyssBossPractice("Abyssus, Heart of the Void", 50, 250, "defensive")
	if !reflect.DeepEqual(balanced, repeat) {
		t.Fatal("identical boss-practice inputs produced different simulations")
	}
	if aggressive.EstimatedDPS <= defensive.EstimatedDPS {
		t.Fatalf("tactics did not change output: aggressive=%d defensive=%d", aggressive.EstimatedDPS, defensive.EstimatedDPS)
	}
	for name, result := range map[string]abyssBossPracticeResult{
		"balanced": balanced, "aggressive": aggressive, "defensive": defensive,
	} {
		if result.Rounds <= 0 || result.Rounds > combatBossEnrageRound(true) || len(result.Logs) < 3 {
			t.Errorf("%s practice result is incomplete: %#v", name, result)
		}
	}
}

func TestAbyssBossPracticeUIExposesSandboxTactics(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{
		"practiceBossName", "Simulation tactic", "aggressive", "defensive",
		"Deterministic sandbox only", "player_max_hp", "boss_max_hp",
	} {
		if !strings.Contains(string(page), token) {
			t.Errorf("practice UI missing %q", token)
		}
	}
}
