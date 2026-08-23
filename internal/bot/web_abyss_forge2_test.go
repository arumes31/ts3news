package bot

import (
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/config"
	"ts3news/internal/content"
)

// captureJSON is a sqlmock.Argument that records the marshalled item_data a
// forge handler persists, so tests can assert what was actually baked in.
type captureJSON struct {
	dst *string
}

func (c captureJSON) Match(v driver.Value) bool {
	if s, ok := v.(string); ok {
		*c.dst = s
	}
	return true
}

// newForge2TestServer returns a WebServer backed by a sqlmock database.
func newForge2TestServer(t *testing.T) (*WebServer, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	b := &Bot{Cfg: &config.Config{}, DB: db}
	return &WebServer{bot: b}, mock, func() { _ = db.Close() }
}

// postForge2 invokes a forge2 handler with a JSON body and returns the response.
func postForge2(t *testing.T, h func(w http.ResponseWriter, r *http.Request, uid string), body, uid string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/abyss/forge2", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h(rec, req, uid)
	return rec
}

// TestAbyssSharpenBakesSTR runs the full sharpen flow on an equipped weapon and
// verifies +2% STR is baked into the persisted item_data.
func TestAbyssSharpenBakesSTR(t *testing.T) {
	s, mock, done := newForge2TestServer(t)
	defer done()
	uid := "forge2sharpen="

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_gear").
		WithArgs("MainHand", uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", nil))
	mock.ExpectQuery("SELECT forge_rep FROM users").
		WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"forge_rep"}).AddRow(0))
	mock.ExpectExec("UPDATE users SET gold = gold -").
		WithArgs(sqlmock.AnyArg(), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE user_materials SET count = count -").
		WithArgs(2, uid, "dust").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE users SET forge_undo=").
		WithArgs(uid, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	var saved string
	mock.ExpectExec("UPDATE user_gear SET item_data=").
		WithArgs(captureJSON{&saved}, "MainHand", uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec("INSERT INTO forge_history").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE users SET forge_rep").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT gold FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(5000))
	mock.ExpectQuery("SELECT mat_id, count FROM user_materials").
		WillReturnRows(sqlmock.NewRows([]string{"mat_id", "count"}))

	rec := postForge2(t, s.handleAbyssSharpen, `{"slot":"MainHand"}`, uid)
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("sharpen failed: %s (pending: %v)", rec.Body.String(), mock.ExpectationsWereMet())
	}
	// U_LEG_2 has STR 1000, so +2% must bake +20.
	if !strings.Contains(rec.Body.String(), "+20 STR") {
		t.Errorf("expected +20 STR in message, got %s", rec.Body.String())
	}
	var g content.Gear
	if err := json.Unmarshal([]byte(saved), &g); err != nil {
		t.Fatalf("saved item_data not parseable: %v", err)
	}
	if g.Stats.STR != 1020 {
		t.Errorf("baked STR = %d, want 1020", g.Stats.STR)
	}
	if g.Sharpened != 1 {
		t.Errorf("Sharpened = %d, want 1", g.Sharpened)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAbyssSharpenRejectsNonWeapon verifies armor cannot be sharpened.
func TestAbyssSharpenRejectsNonWeapon(t *testing.T) {
	s, mock, done := newForge2TestServer(t)
	defer done()
	uid := "forge2sharpen="

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_gear").
		WithArgs("Chest", uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("B_GOOD_2", nil))
	mock.ExpectRollback()

	rec := postForge2(t, s.handleAbyssSharpen, `{"slot":"Chest"}`, uid)
	body := rec.Body.String()
	if strings.Contains(body, `"ok":true`) || !strings.Contains(body, "only weapons") {
		t.Fatalf("expected weapon-only rejection, got %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAbyssReinforceCap verifies reinforcement stops at level 10.
func TestAbyssReinforceCap(t *testing.T) {
	s, mock, done := newForge2TestServer(t)
	defer done()
	uid := "forge2reinforce="

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").
		WithArgs(int64(7), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("B_GOOD_2", `{"reinforced":10}`))
	mock.ExpectRollback()

	rec := postForge2(t, s.handleAbyssReinforce, `{"inv_id":7}`, uid)
	if !strings.Contains(rec.Body.String(), "maximum") {
		t.Fatalf("expected cap rejection, got %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAbyssPolishCapsAtBasePlus100 verifies polish refuses once the cap is hit.
func TestAbyssPolishCapsAtBasePlus100(t *testing.T) {
	s, mock, done := newForge2TestServer(t)
	defer done()
	uid := "forge2polish="

	// B_GOOD_2 catalog MaxDurability is 150, so 250 is the cap.
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").
		WithArgs(int64(3), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("B_GOOD_2", `{"MaxDurability":250}`))
	mock.ExpectRollback()

	rec := postForge2(t, s.handleAbyssPolish, `{"inv_id":3}`, uid)
	if !strings.Contains(rec.Body.String(), "limit") {
		t.Fatalf("expected polish-limit rejection, got %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAbyssAttuneBinds verifies attune bakes +5% stats and sets the flag.
func TestAbyssAttuneBinds(t *testing.T) {
	s, mock, done := newForge2TestServer(t)
	defer done()
	uid := "forge2attune="

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").
		WithArgs(int64(9), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", nil))
	mock.ExpectExec("UPDATE users SET abyss_tokens = abyss_tokens -").
		WithArgs(int64(15), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE users SET forge_undo=").
		WithArgs(uid, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	var saved string
	mock.ExpectExec("UPDATE user_inventory SET item_data=").
		WithArgs(captureJSON{&saved}, int64(9), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec("INSERT INTO forge_history").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE users SET forge_rep").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT abyss_tokens FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"abyss_tokens"}).AddRow(40))

	rec := postForge2(t, s.handleAbyssAttune, `{"inv_id":9}`, uid)
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("attune failed: %s", rec.Body.String())
	}
	var g content.Gear
	if err := json.Unmarshal([]byte(saved), &g); err != nil {
		t.Fatalf("saved item_data not parseable: %v", err)
	}
	if !g.Attuned {
		t.Error("Attuned flag not set in saved item_data")
	}
	if g.Stats.STR != 1050 { // 1000 × 1.05
		t.Errorf("baked STR = %d, want 1050", g.Stats.STR)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAbyssFuseRejectsAttuned verifies an attuned item cannot enter a fusion.
func TestAbyssFuseRejectsAttuned(t *testing.T) {
	s, mock, done := newForge2TestServer(t)
	defer done()
	uid := "forge2fuse="

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").
		WithArgs(int64(1), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", `{"attuned":true}`))
	mock.ExpectRollback()

	rec := postForge2(t, s.handleAbyssFuse, `{"inv_ids":[1,2,3]}`, uid)
	body := rec.Body.String()
	if strings.Contains(body, `"ok":true`) || !strings.Contains(body, "attuned") {
		t.Fatalf("expected attuned rejection, got %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAbyssAwakenRefusesSpecial verifies awaken only touches Special-less items.
func TestAbyssAwakenRefusesSpecial(t *testing.T) {
	s, mock, done := newForge2TestServer(t)
	defer done()
	uid := "forge2awaken="

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").
		WithArgs(int64(4), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", `{"Special":"Thorns"}`))
	mock.ExpectRollback()

	rec := postForge2(t, s.handleAbyssAwaken, `{"inv_id":4}`, uid)
	if !strings.Contains(rec.Body.String(), "no dormant power") {
		t.Fatalf("expected awaken rejection, got %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAbyssImbueRejectsDuplicate verifies an effect already on the item can't be
// imbued again.
func TestAbyssImbueRejectsDuplicate(t *testing.T) {
	s, mock, done := newForge2TestServer(t)
	defer done()
	uid := "forge2imbue="

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").
		WithArgs(int64(5), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", `{"Special":"Vampiric"}`))
	mock.ExpectRollback()

	rec := postForge2(t, s.handleAbyssImbue, `{"inv_id":5,"effect":"vampiric"}`, uid)
	if !strings.Contains(rec.Body.String(), "already carries") {
		t.Fatalf("expected duplicate rejection, got %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAbyssEmbraceRejectsUncorrupted verifies embrace only applies to corrupted gear.
func TestAbyssEmbraceRejectsUncorrupted(t *testing.T) {
	s, mock, done := newForge2TestServer(t)
	defer done()
	uid := "forge2embrace="

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").
		WithArgs(int64(6), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", nil))
	mock.ExpectRollback()

	rec := postForge2(t, s.handleAbyssEmbrace, `{"inv_id":6}`, uid)
	if !strings.Contains(rec.Body.String(), "not corrupted") {
		t.Fatalf("expected not-corrupted rejection, got %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestForge2PoolsAndNames sanity-checks the effect pools and quality ladder.
func TestForge2PoolsAndNames(t *testing.T) {
	seen := map[content.ItemEffect]bool{}
	for _, e := range awakenPool {
		if e == content.EffectNone {
			t.Error("awakenPool contains an empty effect")
		}
		if seen[e] {
			t.Errorf("awakenPool contains duplicate %q", e)
		}
		seen[e] = true
	}
	seen = map[content.ItemEffect]bool{}
	for key, e := range imbueEffects {
		if e == content.EffectNone || seen[e] {
			t.Errorf("imbueEffects[%q] invalid or duplicate %q", key, e)
		}
		seen[e] = true
	}
	if len(qualityNames) != masterworkMax+1 {
		t.Errorf("qualityNames has %d entries, want %d", len(qualityNames), masterworkMax+1)
	}
	if !isWeaponSlot(content.SlotMainHand) || !isWeaponSlot(content.SlotRanged) {
		t.Error("weapon slots not recognised")
	}
	if isWeaponSlot(content.SlotChest) {
		t.Error("Chest must not be a weapon slot")
	}
}

// TestForge2GearFieldsRoundTrip guards the omitempty JSON tags on the new
// per-instance Gear fields: they must survive the item_data round trip.
func TestForge2GearFieldsRoundTrip(t *testing.T) {
	g := content.Gear{
		ID: "X", Name: "Test", Slot: content.SlotChest, Rarity: content.RarityRare,
		Reinforced: 3, Sharpened: 2, Awakened: true, Imbued: "Thorns",
		Attuned: true, Quality: 4, Embraced: true,
	}
	data, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back content.Gear
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Reinforced != 3 || back.Sharpened != 2 || !back.Awakened ||
		back.Imbued != "Thorns" || !back.Attuned || back.Quality != 4 || !back.Embraced {
		t.Errorf("round trip lost fields: %+v", back)
	}
	// Zero values must stay omitted so legacy payloads overlay cleanly.
	empty, _ := json.Marshal(content.Gear{ID: "Y"})
	for _, key := range []string{"reinforced", "sharpened", "awakened", "imbued", "attuned", "quality", "embraced"} {
		if strings.Contains(string(empty), key) {
			t.Errorf("zero-value field %q should be omitted from %s", key, empty)
		}
	}
}
