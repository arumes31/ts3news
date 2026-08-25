package bot

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssRecentlyLootedRejectsFutureTimestamps(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name       string
		acquiredAt time.Time
		want       bool
	}{
		{"just acquired", now, true},
		{"within window", now.Add(-9 * time.Minute), true},
		{"at boundary", now.Add(-abyssRecentlyLootedWindow), true},
		{"too old", now.Add(-abyssRecentlyLootedWindow - time.Nanosecond), false},
		{"future clock skew", now.Add(time.Second), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := abyssRecentlyLooted(test.acquiredAt, now); got != test.want {
				t.Fatalf("abyssRecentlyLooted() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestAbyssInventoryLockIsOwnerScoped(t *testing.T) {
	t.Parallel()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectExec("UPDATE user_inventory SET locked=\\$1 WHERE id=\\$2 AND client_uid=\\$3").
		WithArgs(true, int64(42), "player").
		WillReturnResult(sqlmock.NewResult(0, 1))
	server := &WebServer{bot: &Bot{DB: database}}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/inventory/lock", strings.NewReader(`{"inv_id":42,"locked":true}`))
	server.handleAbyssInventoryLock(response, request, "player")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"locked":true`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssInventoryLockRejectsInvalidRequests(t *testing.T) {
	t.Parallel()
	server := &WebServer{bot: &Bot{}}
	for _, test := range []struct {
		method string
		body   string
		code   int
	}{
		{http.MethodGet, ``, http.StatusMethodNotAllowed},
		{http.MethodPost, `{"inv_id":0}`, http.StatusOK},
		{http.MethodPost, `{"inv_id":-4}`, http.StatusOK},
	} {
		response := httptest.NewRecorder()
		server.handleAbyssInventoryLock(response, httptest.NewRequest(test.method, "/api/abyss/inventory/lock", strings.NewReader(test.body)), "player")
		if response.Code != test.code {
			t.Errorf("%s %s: status %d", test.method, test.body, response.Code)
		}
	}
}

func TestAbyssPanelToolsContracts(t *testing.T) {
	t.Parallel()
	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if server.tmpl.Lookup("abyssPanelToolsJS") == nil {
		t.Fatal("panel tools script template is missing")
	}
	var source strings.Builder
	for _, asset := range []string{"webassets/abyss.html", "webassets/abyss_panel_tools.html", "webassets/abyss_panel_tools.css"} {
		raw, err := webAssets.ReadFile(asset)
		if err != nil {
			t.Fatalf("read %s: %v", asset, err)
		}
		source.Write(raw)
	}
	for _, token := range []string{
		"/api/abyss/inventory/lock", "ab_backpack_pins", "ab-bp-select", "Salvage selected",
		"abPinnedCompare", "abArmouryTotal", "abRecipeProgress", "abForgeSearch",
		"abStockClock", "abNearestAchievement", "ab-tab-points", "ab-side-tuning",
		"data-locked", "data-recent", "data-stats",
	} {
		if !strings.Contains(source.String(), token) {
			t.Errorf("Abyss panel tools are missing %q", token)
		}
	}
}
