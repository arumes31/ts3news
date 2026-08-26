package bot

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

type transmogGearJSONMatcher struct {
	want content.Gear
}

func (matcher transmogGearJSONMatcher) Match(value driver.Value) bool {
	var raw []byte
	switch typed := value.(type) {
	case string:
		raw = []byte(typed)
	case []byte:
		raw = typed
	default:
		return false
	}
	var got content.Gear
	return json.Unmarshal(raw, &got) == nil && reflect.DeepEqual(got, matcher.want)
}

func transmogTestPair(t *testing.T) (content.Gear, content.Gear) {
	t.Helper()
	catalog := content.GearAppearanceCatalog()
	for i, target := range catalog {
		for _, appearance := range catalog[i+1:] {
			if target.Slot == appearance.Slot && target.ID != appearance.ID {
				return target, appearance
			}
		}
	}
	t.Fatal("gear catalog has no same-slot appearance pair")
	return content.Gear{}, content.Gear{}
}

func TestAbyssTransmogCostScalesByRarity(t *testing.T) {
	t.Parallel()
	if abyssTransmogCost(content.RarityCommon) != 10_000 {
		t.Fatal("common transmog cost changed")
	}
	if abyssTransmogCost(content.RarityEternal) <= abyssTransmogCost(content.RarityLegendary) {
		t.Fatal("top-tier transmog must cost more than legendary")
	}
	if abyssTransmogCost(content.Rarity(99)) != abyssTransmogCost(content.RarityEternal) {
		t.Fatal("invalid rarity was not capped")
	}
}

func TestAbyssTransmogViewsIgnoreForeignAndStaleCosmetics(t *testing.T) {
	t.Parallel()
	target, _ := transmogTestPair(t)
	owned := map[string]bool{
		abyssTransmogKey(target.ID): true,
		abyssTransmogKey("missing"): true,
		"boss_banner_crownless":     true,
	}
	views := abyssTransmogViews(abyssTransmogCatalog(), owned)
	if len(views) != 1 || views[0].ID != target.ID || views[0].Slot != string(target.Slot) {
		t.Fatalf("filtered transmog views = %#v", views)
	}
}

func TestGearViewExposesOnlyCompatibleCatalogAppearance(t *testing.T) {
	t.Parallel()
	target, appearance := transmogTestPair(t)
	target.AppearanceID = appearance.ID
	view := toGearView(target.Slot, target)
	if view.AppearanceID != appearance.ID || view.AppearanceName != appearance.Name {
		t.Fatalf("compatible appearance view = %#v", view)
	}
	for _, gear := range content.GearAppearanceCatalog() {
		if gear.Slot == target.Slot {
			continue
		}
		target.AppearanceID = gear.ID
		view = toGearView(target.Slot, target)
		if view.AppearanceID != "" || view.AppearanceName != "" {
			t.Fatalf("cross-slot appearance leaked into view: %#v", view)
		}
		return
	}
	t.Fatal("gear catalog has no incompatible slot")
}

func TestSyncAbyssTransmogUnlocksUsesIdentifiedCatalogGearOnly(t *testing.T) {
	t.Parallel()
	target, appearance := transmogTestPair(t)
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	targetJSON, _ := json.Marshal(target)
	appearance.Unidentified = true
	unidentifiedJSON, _ := json.Marshal(appearance)
	unknownJSON, _ := json.Marshal(content.Gear{ID: "unknown", Slot: target.Slot, Name: "Unknown"})

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id,item_data FROM user_gear").WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).
			AddRow(target.ID, string(targetJSON)).
			AddRow(appearance.ID, string(unidentifiedJSON)).
			AddRow("unknown", string(unknownJSON)))
	mock.ExpectExec("INSERT INTO abyss_shop_cosmetics").
		WithArgs("user", abyssTransmogKey(target.ID)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := database.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	unlocked, err := (&Bot{DB: database}).syncAbyssTransmogUnlocks(tx, "user")
	if err != nil {
		t.Fatalf("sync unlocks: %v", err)
	}
	if unlocked != 1 {
		t.Fatalf("new unlocks = %d, want 1", unlocked)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssTransmogApplyIsAtomicAndCosmeticOnly(t *testing.T) {
	t.Parallel()
	target, appearance := transmogTestPair(t)
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	targetJSON, _ := json.Marshal(target)
	want := target
	want.AppearanceID = appearance.ID
	cost := abyssTransmogCost(appearance.Rarity)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory.*FOR UPDATE").
		WithArgs(int64(7), "user").
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow(target.ID, string(targetJSON)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM abyss_shop_cosmetics")).
		WithArgs("user", abyssTransmogKey(appearance.ID)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec("UPDATE user_inventory SET item_data").
		WithArgs(transmogGearJSONMatcher{want: want}, int64(7), "user").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE users SET gold = gold -").WithArgs(cost, "user").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT gold FROM users").WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(900_000))
	mock.ExpectCommit()

	server := &WebServer{bot: &Bot{DB: database}}
	body := `{"inv_id":7,"appearance_id":"` + appearance.ID + `"}`
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/transmog/apply", bytes.NewBufferString(body))
	response := httptest.NewRecorder()
	server.handleAbyssTransmogApply(response, request, "user")
	got := response.Body.String()
	if !strings.Contains(got, `"ok":true`) ||
		!strings.Contains(got, `"appearance_id":"`+appearance.ID+`"`) {
		t.Fatalf("apply response = %s", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssTransmogInsufficientGoldRollsBackItemRewrite(t *testing.T) {
	t.Parallel()
	target, appearance := transmogTestPair(t)
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	targetJSON, _ := json.Marshal(target)
	want := target
	want.AppearanceID = appearance.ID

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory.*FOR UPDATE").
		WithArgs(int64(9), "user").
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow(target.ID, string(targetJSON)))
	mock.ExpectQuery("SELECT EXISTS").WithArgs("user", abyssTransmogKey(appearance.ID)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec("UPDATE user_inventory SET item_data").
		WithArgs(transmogGearJSONMatcher{want: want}, int64(9), "user").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE users SET gold = gold -").
		WithArgs(abyssTransmogCost(appearance.Rarity), "user").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	server := &WebServer{bot: &Bot{DB: database}}
	body := `{"inv_id":9,"appearance_id":"` + appearance.ID + `"}`
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/transmog/apply", bytes.NewBufferString(body))
	response := httptest.NewRecorder()
	server.handleAbyssTransmogApply(response, request, "user")
	if !strings.Contains(response.Body.String(), "not enough gold") {
		t.Fatalf("insufficient-gold response = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssTransmogClearIsFree(t *testing.T) {
	t.Parallel()
	target, appearance := transmogTestPair(t)
	target.AppearanceID = appearance.ID
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	targetJSON, _ := json.Marshal(target)
	want := target
	want.AppearanceID = ""

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_gear.*FOR UPDATE").
		WithArgs(string(target.Slot), "user").
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow(target.ID, string(targetJSON)))
	mock.ExpectExec("UPDATE user_gear SET item_data").
		WithArgs(transmogGearJSONMatcher{want: want}, string(target.Slot), "user").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT gold FROM users").WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(42_000))
	mock.ExpectCommit()

	server := &WebServer{bot: &Bot{DB: database}}
	body := `{"slot":"` + string(target.Slot) + `","appearance_id":""}`
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/transmog/apply", bytes.NewBufferString(body))
	response := httptest.NewRecorder()
	server.handleAbyssTransmogApply(response, request, "user")
	got := response.Body.String()
	if !strings.Contains(got, `"cost":0`) ||
		!strings.Contains(got, "Original appearance restored") {
		t.Fatalf("clear response = %s", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssTransmogRejectsCrossSlotAppearanceBeforeCharge(t *testing.T) {
	t.Parallel()
	target, _ := transmogTestPair(t)
	var incompatible content.Gear
	for _, gear := range content.GearAppearanceCatalog() {
		if gear.Slot != target.Slot {
			incompatible = gear
			break
		}
	}
	if incompatible.ID == "" {
		t.Fatal("gear catalog has no incompatible slot")
	}
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	targetJSON, _ := json.Marshal(target)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory.*FOR UPDATE").
		WithArgs(int64(12), "user").
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow(target.ID, string(targetJSON)))
	mock.ExpectRollback()

	server := &WebServer{bot: &Bot{DB: database}}
	body := `{"inv_id":12,"appearance_id":"` + incompatible.ID + `"}`
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/transmog/apply", bytes.NewBufferString(body))
	response := httptest.NewRecorder()
	server.handleAbyssTransmogApply(response, request, "user")
	if !strings.Contains(response.Body.String(), "incompatible appearance") {
		t.Fatalf("cross-slot response = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssTransmogAssetsAndRoutes(t *testing.T) {
	t.Parallel()
	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	for _, name := range []string{"abyssTransmogPanel", "abyssTransmogJS"} {
		if server.tmpl.Lookup(name) == nil {
			t.Fatalf("missing template %q", name)
		}
	}
	page, _ := webAssets.ReadFile("webassets/abyss.html")
	css, _ := webAssets.ReadFile("webassets/abyss_transmog.css")
	routes, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatalf("read routes: %v", err)
	}
	for _, required := range []string{
		`{{template "abyssTransmogPanel" .}}`, `{{template "abyssTransmogJS" .}}`,
		`{{asset "/static/abyss_transmog.css"}}`, `data-appearance-id="{{.AppearanceID}}"`,
	} {
		if !strings.Contains(string(page), required) {
			t.Errorf("Abyss page is missing %q", required)
		}
	}
	for _, required := range []string{".ab-wardrobe-grid", "@media(max-width:720px)", "@media(forced-colors:active)"} {
		if !strings.Contains(string(css), required) {
			t.Errorf("transmog CSS is missing %q", required)
		}
	}
	for _, required := range []string{
		`/static/abyss_transmog.css`, `/api/abyss/transmog`,
		`/api/abyss/transmog/apply`, `s.authAPI(s.handleAbyssTransmogApply)`,
	} {
		if !strings.Contains(string(routes), required) {
			t.Errorf("web routes are missing %q", required)
		}
	}
}
