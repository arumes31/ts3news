package bot

import (
	"database/sql/driver"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/config"
	"ts3news/internal/content"
)

// captureJSON3 mirrors the forge2 test matcher: records persisted item_data.
type captureJSON3 struct {
	dst *string
}

func (c captureJSON3) Match(v driver.Value) bool {
	if s, ok := v.(string); ok {
		*c.dst = s
	}
	return true
}

func newForge3TestServer(t *testing.T) (*WebServer, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	b := &Bot{Cfg: &config.Config{}, DB: db}
	return &WebServer{bot: b}, mock, func() { _ = db.Close() }
}

// ---- Rarities -----------------------------------------------------------------

// TestCelestialEternalRarityPlumbing verifies the two new top rarities are wired
// into naming, colors, combat rating, upgrade costs, dismantle yields and
// material yields.
func TestCelestialEternalRarityPlumbing(t *testing.T) {
	if content.RarityCelestial <= content.RarityDivine || content.RarityEternal <= content.RarityCelestial {
		t.Fatal("rarity order must be Divine < Celestial < Eternal")
	}
	if content.RarityCelestial.String() == content.RarityDivine.String() {
		t.Error("Celestial needs its own name")
	}
	if content.RarityCelestial.Color() == content.RarityDivine.Color() ||
		content.RarityEternal.Color() == content.RarityCelestial.Color() {
		t.Error("new rarities need distinct colors")
	}
	base := content.Gear{Rarity: content.RarityDivine, Stats: content.Stats{STR: 100}}
	cel := base
	cel.Rarity = content.RarityCelestial
	ete := base
	ete.Rarity = content.RarityEternal
	if !(base.CombatRating() < cel.CombatRating() && cel.CombatRating() < ete.CombatRating()) {
		t.Errorf("CR must rise with rarity: %.0f / %.0f / %.0f", base.CombatRating(), cel.CombatRating(), ete.CombatRating())
	}
	if abyssUpgradeGearCost(content.RarityCelestial) != 400 || abyssUpgradeGearCost(content.RarityEternal) != 800 {
		t.Error("upgrade costs for Celestial/Eternal wrong")
	}
	if abyssDismantleTokens(content.RarityCelestial) != 15 || abyssDismantleTokens(content.RarityEternal) != 25 {
		t.Error("dismantle token yields for Celestial/Eternal wrong")
	}
	if m, n := materialYieldForRarity(content.RarityCelestial); m != "prism" || n != 2 {
		t.Errorf("Celestial material yield = %s x%d, want prism x2", m, n)
	}
	if m, n := materialYieldForRarity(content.RarityEternal); m != "prism" || n != 3 {
		t.Errorf("Eternal material yield = %s x%d, want prism x3", m, n)
	}
}

// TestSetBonusExtensions covers the new 8-piece legacy tier and the harvester set.
func TestSetBonusExtensions(t *testing.T) {
	if _, r := content.AbyssSetBonus(8); r != 8 {
		t.Errorf("8 legacy pieces should reach the 8-piece tier, got %d", r)
	}
	bonus, reached := content.AbyssSetBonusBySet(map[string]int{"harvester": 3})
	if reached["harvester"] != 3 {
		t.Errorf("3 harvester pieces should reach the 3-piece tier, got %d", reached["harvester"])
	}
	if bonus.LCK != 60+150 {
		t.Errorf("harvester 2+3 piece LCK = %d, want %d", bonus.LCK, 210)
	}
	// The harvester items exist and carry the set tag.
	for _, id := range []string{"ABYSS_HARVESTER_HOOK", "ABYSS_HARVESTER_EYE", "ABYSS_HARVESTER_BAND"} {
		g, ok := content.GetGearByID(id)
		if !ok {
			t.Fatalf("harvester item %s missing from catalog", id)
		}
		if g.EffectiveSetID() != "harvester" {
			t.Errorf("%s set = %q, want harvester", id, g.EffectiveSetID())
		}
	}
}

// ---- Forge 4 handlers -----------------------------------------------------------

// TestAbyssRebalanceMovesStat runs the full rebalance flow and verifies 25% of
// STR lands in DEF.
func TestAbyssRebalanceMovesStat(t *testing.T) {
	s, mock, done := newForge3TestServer(t)
	defer done()
	uid := "forge3rebalance="

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").
		WithArgs(int64(11), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", nil))
	mock.ExpectQuery("SELECT forge_rep FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"forge_rep"}).AddRow(0))
	mock.ExpectExec("UPDATE users SET gold = gold -").
		WithArgs(sqlmock.AnyArg(), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE users SET forge_undo=").
		WillReturnResult(sqlmock.NewResult(0, 1))
	var saved string
	mock.ExpectExec("UPDATE user_inventory SET item_data=").
		WithArgs(captureJSON3{&saved}, int64(11), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec("INSERT INTO forge_history").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE users SET forge_rep").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT gold FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(1000))

	req := httptest.NewRequest("POST", "/api/abyss/rebalance", strings.NewReader(`{"inv_id":11,"from":"STR","to":"DEF"}`))
	rec := httptest.NewRecorder()
	s.handleAbyssRebalance(rec, req, uid)
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("rebalance failed: %s", rec.Body.String())
	}
	var g content.Gear
	if err := json.Unmarshal([]byte(saved), &g); err != nil {
		t.Fatalf("saved item_data not parseable: %v", err)
	}
	// U_LEG_2 has STR 1000, DEF 0: 25% of 1000 = 250 moves over.
	if g.Stats.STR != 750 || g.Stats.DEF != 250 {
		t.Errorf("after rebalance STR=%d DEF=%d, want 750/250", g.Stats.STR, g.Stats.DEF)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAbyssRebalanceSameStatRejected verifies from==to never touches the DB.
func TestAbyssRebalanceSameStatRejected(t *testing.T) {
	s, mock, done := newForge3TestServer(t)
	defer done()
	req := httptest.NewRequest("POST", "/api/abyss/rebalance", strings.NewReader(`{"inv_id":11,"from":"STR","to":"str"}`))
	rec := httptest.NewRecorder()
	s.handleAbyssRebalance(rec, req, "forge3rebalance=")
	if !strings.Contains(rec.Body.String(), "different stats") {
		t.Fatalf("expected same-stat rejection, got %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("no DB calls expected: %v", err)
	}
}

// TestAbyssGemTransmuteSwapsStats verifies the first gem's stat line is swapped
// at the same tier.
func TestAbyssGemTransmuteSwapsStats(t *testing.T) {
	s, mock, done := newForge3TestServer(t)
	defer done()
	uid := "forge3gem="

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").
		WithArgs(int64(12), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", `{"Gemstones":["Ruby II"]}`))
	mock.ExpectQuery("SELECT forge_rep FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"forge_rep"}).AddRow(0))
	mock.ExpectExec("UPDATE users SET gold = gold -").
		WithArgs(sqlmock.AnyArg(), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE users SET forge_undo=").
		WillReturnResult(sqlmock.NewResult(0, 1))
	var saved string
	mock.ExpectExec("UPDATE user_inventory SET item_data=").
		WithArgs(captureJSON3{&saved}, int64(12), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec("INSERT INTO forge_history").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE users SET forge_rep").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT gold FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(1000))

	req := httptest.NewRequest("POST", "/api/abyss/transmute_gem", strings.NewReader(`{"inv_id":12,"gem":"Diamond"}`))
	rec := httptest.NewRecorder()
	s.handleAbyssGemTransmute(rec, req, uid)
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("gem transmute failed: %s", rec.Body.String())
	}
	var g content.Gear
	if err := json.Unmarshal([]byte(saved), &g); err != nil {
		t.Fatalf("saved item_data not parseable: %v", err)
	}
	if len(g.Gemstones) != 1 || g.Gemstones[0] != "Diamond II" {
		t.Errorf("gemstones = %v, want [Diamond II]", g.Gemstones)
	}
	// Ruby II baked +200 HP; U_LEG_2 has 0 HP → −200 after removal. Diamond II adds +30 DEF.
	if g.Stats.HP != -200 || g.Stats.DEF != 30 {
		t.Errorf("stats after transmute HP=%d DEF=%d, want −200/30", g.Stats.HP, g.Stats.DEF)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAbyssSwapSpecial verifies two items exchange their Specials.
func TestAbyssSwapSpecial(t *testing.T) {
	s, mock, done := newForge3TestServer(t)
	defer done()
	uid := "forge3swap="

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").
		WithArgs(int64(13), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_6", nil))
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").
		WithArgs(int64(14), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_7", nil))
	mock.ExpectExec("UPDATE user_materials SET count = count -").
		WithArgs(8, uid, "core").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE users SET forge_undo=").
		WillReturnResult(sqlmock.NewResult(0, 1))
	var saved1, saved2 string
	mock.ExpectExec("UPDATE user_inventory SET item_data=").
		WithArgs(captureJSON3{&saved1}, int64(13), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE user_inventory SET item_data=").
		WithArgs(captureJSON3{&saved2}, int64(14), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec("INSERT INTO forge_history").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE users SET forge_rep").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT mat_id, count FROM user_materials").
		WillReturnRows(sqlmock.NewRows([]string{"mat_id", "count"}))

	req := httptest.NewRequest("POST", "/api/abyss/swap_special", strings.NewReader(`{"inv_id":13,"inv_id2":14}`))
	rec := httptest.NewRecorder()
	s.handleAbyssSwapSpecial(rec, req, uid)
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("swap failed: %s", rec.Body.String())
	}
	var g1, g2 content.Gear
	if err := json.Unmarshal([]byte(saved1), &g1); err != nil {
		t.Fatalf("saved item 1 not parseable: %v", err)
	}
	if err := json.Unmarshal([]byte(saved2), &g2); err != nil {
		t.Fatalf("saved item 2 not parseable: %v", err)
	}
	// U_LEG_6 has Vampiric, U_LEG_7 has Thorns — they must trade places.
	if g1.Special != content.EffectThorns || g2.Special != content.EffectVampiric {
		t.Errorf("after swap: %s has %q, %s has %q", g1.Name, g1.Special, g2.Name, g2.Special)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAbyssTemperSurgeRequiresMaxTemper verifies surging needs +15 first.
func TestAbyssTemperSurgeRequiresMaxTemper(t *testing.T) {
	s, mock, done := newForge3TestServer(t)
	defer done()
	uid := "forge3surge="

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").
		WithArgs(int64(15), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", `{"temper":10}`))
	mock.ExpectRollback()

	req := httptest.NewRequest("POST", "/api/abyss/temper_surge", strings.NewReader(`{"inv_id":15}`))
	rec := httptest.NewRecorder()
	s.handleAbyssTemperSurge(rec, req, uid)
	if !strings.Contains(rec.Body.String(), "temper the item to +15 first") {
		t.Fatalf("expected temper prerequisite, got %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAbyssCorruptRejectsCorrupted verifies double-corruption is refused.
func TestAbyssCorruptRejectsCorrupted(t *testing.T) {
	s, mock, done := newForge3TestServer(t)
	defer done()
	uid := "forge3corrupt="

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").
		WithArgs(int64(16), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", `{"corrupted":true}`))
	mock.ExpectRollback()

	req := httptest.NewRequest("POST", "/api/abyss/corrupt", strings.NewReader(`{"inv_id":16}`))
	rec := httptest.NewRecorder()
	s.handleAbyssCorrupt(rec, req, uid)
	if !strings.Contains(rec.Body.String(), "already corrupted") {
		t.Fatalf("expected already-corrupted rejection, got %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAbyssPrismaticRequiresRune verifies the rune prerequisite.
func TestAbyssPrismaticRequiresRune(t *testing.T) {
	s, mock, done := newForge3TestServer(t)
	defer done()
	uid := "forge3prismatic="

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").
		WithArgs(int64(17), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", nil))
	mock.ExpectRollback()

	req := httptest.NewRequest("POST", "/api/abyss/prismatic_rune", strings.NewReader(`{"inv_id":17}`))
	rec := httptest.NewRecorder()
	s.handleAbyssPrismaticRune(rec, req, uid)
	if !strings.Contains(rec.Body.String(), "etch a rune first") {
		t.Fatalf("expected rune prerequisite, got %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAbyssBrandUnknownSet verifies the set whitelist is enforced before any DB work.
func TestAbyssBrandUnknownSet(t *testing.T) {
	s, mock, done := newForge3TestServer(t)
	defer done()
	req := httptest.NewRequest("POST", "/api/abyss/brand", strings.NewReader(`{"inv_id":18,"set":"dragon"}`))
	rec := httptest.NewRecorder()
	s.handleAbyssBrand(rec, req, "forge3brand=")
	if !strings.Contains(rec.Body.String(), "unknown set") {
		t.Fatalf("expected unknown-set rejection, got %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("no DB calls expected: %v", err)
	}
}

// TestAbyssInfuseXPRejectsNonWeapon verifies gear-XP infusion targets weapons only.
func TestAbyssInfuseXPRejectsNonWeapon(t *testing.T) {
	s, mock, done := newForge3TestServer(t)
	defer done()
	uid := "forge3infuse="

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_gear").
		WithArgs("Chest", uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("B_GOOD_2", nil))
	mock.ExpectRollback()

	req := httptest.NewRequest("POST", "/api/abyss/infuse_xp", strings.NewReader(`{"slot":"Chest","sacrifice_inv_id":19}`))
	rec := httptest.NewRecorder()
	s.handleAbyssInfuseXP(rec, req, uid)
	if !strings.Contains(rec.Body.String(), "only weapons") {
		t.Fatalf("expected weapon-only rejection, got %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestForge3Helpers sanity-checks the small pure helpers.
func TestForge3Helpers(t *testing.T) {
	if _, ok := abyssGemBaseStats("Ruby"); !ok {
		t.Error("Ruby base stats missing")
	}
	if _, ok := abyssGemBaseStats("Mithril"); ok {
		t.Error("unknown gem should not resolve")
	}
	if abyssGemTierMultiplier("II") != 2 || abyssGemTierMultiplier("III") != 4 || abyssGemTierMultiplier("") != 1 {
		t.Error("gem tier multipliers wrong")
	}
	var st content.Stats
	st.STR = 10
	*gearStatRef(&st, "STR") += 5
	if st.STR != 15 {
		t.Error("gearStatRef did not mutate STR")
	}
	if gearStatRef(&st, "CHA") != nil {
		t.Error("flavour stats must not be rebalancesable")
	}
	if !brandableSets["harvester"] || brandableSets["abyss_legacy"] {
		t.Error("brandable set whitelist wrong")
	}
}
