package bot

import (
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssUI161Through170Contracts(t *testing.T) {
	t.Parallel()

	mobileBytes, err := webAssets.ReadFile("webassets/abyss_mobile.html")
	if err != nil {
		t.Fatal(err)
	}
	cssBytes, err := webAssets.ReadFile("webassets/abyss_mobile.css")
	if err != nil {
		t.Fatal(err)
	}
	pageBytes, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	mobile, css, page := string(mobileBytes), string(cssBytes), string(pageBytes)

	for _, contract := range []string{
		`#abyssForgePanel details.ab-acc`, `select.id='abyssMobileTier'`,
		`initLandscapeLog`, `initBackpackKeyboard`, `event.key==='Enter'`,
		`event.key==='Delete'`, `/api/abyss/salvage`, `/api/inventory/equip`,
		`initRunModalFocus`, `btnDescend`,
	} {
		if !strings.Contains(mobile, contract) {
			t.Errorf("Abyss mobile layer missing contract %q", contract)
		}
	}
	for _, contract := range []string{
		`scroll-snap-type: x mandatory`, `min-height: 36px`,
		`orientation: landscape`, `.ab-landscape-log`, `max-width: 1100px`,
		`overflow-wrap: anywhere`, `prefers-color-scheme: light`, `color-scheme: light`,
	} {
		if !strings.Contains(css, contract) {
			t.Errorf("Abyss mobile stylesheet missing contract %q", contract)
		}
	}
	for _, contract := range []string{
		`/static/abyss_mobile.css`, `aria-describedby="abyssLayoutGuide"`,
		`id="abyssLayoutGuide"`, `{{template "abyss-mobile" .}}`,
		`Enter to equip, Delete to salvage`,
	} {
		if !strings.Contains(page, contract) {
			t.Errorf("Abyss page missing mobile integration contract %q", contract)
		}
	}
}

func TestAbyssTargetedSalvageRejectsProtectedRarity(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM app_meta WHERE key=$1")).
		WithArgs(abyssLootReservedKey("player")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, gear_id, item_data FROM user_inventory WHERE client_uid=$1 AND locked=FALSE AND id=$2")).
		WithArgs("player", int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "gear_id", "item_data"}).AddRow(42, "U_LEG_2", nil))

	request := httptest.NewRequest("POST", "/api/abyss/salvage", strings.NewReader(`{"inv_id":42}`))
	response := httptest.NewRecorder()
	server := &WebServer{bot: &Bot{DB: database}}
	server.handleAbyssSalvage(response, request, "player")

	if body := response.Body.String(); !strings.Contains(body, `"ok":false`) || !strings.Contains(body, "only unreserved Common or Uncommon") {
		t.Fatalf("protected-rarity response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
