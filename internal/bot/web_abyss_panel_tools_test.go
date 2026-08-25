package bot

import (
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssInventoryLockUpdatesOnlyOwnedItem(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	mock.ExpectExec(regexp.QuoteMeta("UPDATE user_inventory SET locked=$1 WHERE id=$2 AND client_uid=$3")).
		WithArgs(true, int64(17), "player").
		WillReturnResult(sqlmock.NewResult(0, 1))

	request := httptest.NewRequest("POST", "/api/abyss/inventory/lock", strings.NewReader(`{"inv_id":17,"locked":true}`))
	recorder := httptest.NewRecorder()
	server := &WebServer{bot: &Bot{DB: database}}
	server.handleAbyssInventoryLock(recorder, request, "player")
	if body := recorder.Body.String(); !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"locked":true`) {
		t.Fatalf("lock response = %q", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssInventoryLockRejectsInvalidIDWithoutDatabaseWrite(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	request := httptest.NewRequest("POST", "/api/abyss/inventory/lock", strings.NewReader(`{"inv_id":0,"locked":true}`))
	recorder := httptest.NewRecorder()
	server := &WebServer{bot: &Bot{DB: database}}
	server.handleAbyssInventoryLock(recorder, request, "player")
	if body := recorder.Body.String(); !strings.Contains(body, "invalid inventory item") {
		t.Fatalf("invalid lock response = %q", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssPanelToolsExtendedContracts(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if server.tmpl.Lookup("abyssPanelToolsJS") == nil {
		t.Fatal("panel tools template is missing")
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	module, err := webAssets.ReadFile("webassets/abyss_panel_tools.html")
	if err != nil {
		t.Fatal(err)
	}
	css, err := webAssets.ReadFile("webassets/abyss_panel_tools.css")
	if err != nil {
		t.Fatal(err)
	}
	source := string(page) + string(module) + string(css)
	for _, required := range []string{
		"ab_inventory_view", "Search forge actions", "ab_recent_forge_actions",
		"ab-bp-select", "/api/abyss/inventory/lock", "ab_backpack_pins",
		"ab-set-progress", "is-low", "ab_next_run_loadout", "inv_ids:ids",
		"Price note", "Daily stock refresh", "Recipe discoveries", "ab_lore_read",
		"Nearest achievement", "Weekly bounty claims left", "ab-tab-points",
		"ab-shop-insanity", "ab-afford-dot", "is-recent", "ab-expiry-warning",
		"ab_side_density", "ab_side_opacity", "data-recipe-state", "data-recent",
		`{{asset "/static/abyss_panel_tools.css"}}`, `{{template "abyssPanelToolsJS" .}}`,
	} {
		if !strings.Contains(source, required) {
			t.Errorf("panel tools are missing %q", required)
		}
	}
	webSource, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(webSource), "/static/abyss_panel_tools.css") || !strings.Contains(string(webSource), "/api/abyss/inventory/lock") {
		t.Error("panel tools routes are not registered")
	}
}

func TestAbyssDestructiveInventoryPathsHonorPersistentLocks(t *testing.T) {
	t.Parallel()

	files := []string{
		"web_pages.go", "web_ah.go", "web_abyss_econ.go", "web_abyss_features.go",
		"web_abyss_forge3.go", "web_abyss_forge_crafting.go", "web_abyss_set_trade.go",
	}
	for _, name := range files {
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(string(source), "\n") {
			if strings.Contains(line, "DELETE FROM user_inventory") && !strings.Contains(line, "locked=FALSE") {
				if name == "web_pages.go" || strings.Contains(name, "loadouts") {
					continue // equipping moves an item; it is not a destructive disposal action.
				}
				t.Errorf("%s has an unguarded destructive inventory delete: %s", name, strings.TrimSpace(line))
			}
		}
	}
}
