package bot

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestLoadAbyssPublicStatsAggregatesCachesAndOrdersTiers(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	expectAbyssPublicStatsQueries(mock)

	server := &WebServer{bot: &Bot{DB: database}}
	now := time.Date(2026, time.August, 25, 12, 34, 56, 789, time.UTC)
	first, err := server.loadAbyssPublicStats(t.Context(), now)
	if err != nil {
		t.Fatalf("loadAbyssPublicStats: %v", err)
	}
	if !first.OK || first.Version != 1 || first.Season != "August 2026" {
		t.Fatalf("metadata = %+v", first)
	}
	if first.GeneratedAt.Nanosecond() != 0 {
		t.Fatalf("generated_at = %v, want whole-second precision", first.GeneratedAt)
	}
	wantTotals := abyssPublicStatsTotals{
		Runs: 12, Delvers: 5, ActiveRuns: 2, FloorsCleared: 84,
		DeepestFloor: 37, GoldBanked: 9000, BankedRuns: 7, FailedRuns: 5,
	}
	if first.Totals != wantTotals {
		t.Fatalf("totals = %+v, want %+v", first.Totals, wantTotals)
	}
	wantTiers := []string{"normal", "hell", "legacy"}
	for index, want := range wantTiers {
		if first.Tiers[index].Tier != want {
			t.Fatalf("tiers[%d] = %q, want %q", index, first.Tiers[index].Tier, want)
		}
	}

	first.Tiers[0].Tier = "mutated"
	second, err := server.loadAbyssPublicStats(t.Context(), now.Add(30*time.Second))
	if err != nil {
		t.Fatalf("cached loadAbyssPublicStats: %v", err)
	}
	if second.Tiers[0].Tier != "normal" {
		t.Fatalf("cache shared mutable tier slice: %+v", second.Tiers)
	}
	encoded, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal stats: %v", err)
	}
	for _, privateField := range []string{"client_uid", "nickname", "history", "combat_log"} {
		if strings.Contains(string(encoded), privateField) {
			t.Errorf("public payload contains private field %q", privateField)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAbyssPublicStatsSupportsHTTPValidation(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	expectAbyssPublicStatsQueries(mock)
	server := &WebServer{bot: &Bot{DB: database}}

	get := httptest.NewRequest(http.MethodGet, "/api/abyss/public/stats", nil)
	response := httptest.NewRecorder()
	server.handleAbyssPublicStats(response, get)
	if response.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body = %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("CORS header = %q", got)
	}
	if got := response.Header().Get("Cache-Control"); got != "public, max-age=60, stale-while-revalidate=300" {
		t.Errorf("Cache-Control = %q", got)
	}
	if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
	etag := response.Header().Get("ETag")
	if etag == "" {
		t.Fatal("GET response has no ETag")
	}

	conditional := httptest.NewRequest(http.MethodGet, "/api/abyss/public/stats", nil)
	conditional.Header.Set("If-None-Match", `"other", `+etag)
	notModified := httptest.NewRecorder()
	server.handleAbyssPublicStats(notModified, conditional)
	if notModified.Code != http.StatusNotModified || notModified.Body.Len() != 0 {
		t.Fatalf("conditional GET = status %d body %q", notModified.Code, notModified.Body.String())
	}

	head := httptest.NewRequest(http.MethodHead, "/api/abyss/public/stats", nil)
	headResponse := httptest.NewRecorder()
	server.handleAbyssPublicStats(headResponse, head)
	if headResponse.Code != http.StatusOK || headResponse.Body.Len() != 0 {
		t.Fatalf("HEAD = status %d body %q", headResponse.Code, headResponse.Body.String())
	}
	if headResponse.Header().Get("ETag") != etag {
		t.Error("HEAD did not return the cached representation ETag")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAbyssPublicStatsRejectsUnsupportedMethods(t *testing.T) {
	server := &WebServer{}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/public/stats", nil)
	response := httptest.NewRecorder()
	server.handleAbyssPublicStats(response, request)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if got := response.Header().Get("Allow"); got != "GET, HEAD" {
		t.Fatalf("Allow = %q", got)
	}
}

func TestHandleAbyssPublicStatsReturnsGenericDatabaseFailure(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT COUNT\\(\\*\\), COUNT\\(DISTINCT client_uid\\)").
		WillReturnError(errors.New("private database details"))
	mock.ExpectRollback()

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodGet, "/api/abyss/public/stats", nil)
	response := httptest.NewRecorder()
	server.handleAbyssPublicStats(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if strings.Contains(response.Body.String(), "private database details") {
		t.Fatalf("response leaked database error: %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssPublicStatsRouteIsAnonymous(t *testing.T) {
	root := abyssAAARepositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(root, "internal", "bot", "web.go"))
	if err != nil {
		t.Fatalf("read web.go: %v", err)
	}
	plainRoute := regexp.MustCompile(
		`mux\.HandleFunc\("/api/abyss/public/stats", s\.handleAbyssPublicStats\)`,
	)
	if !plainRoute.Match(source) {
		t.Fatal("public stats route is absent or wrapped in authentication")
	}
}

func expectAbyssPublicStatsQueries(mock sqlmock.Sqlmock) {
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT COUNT\\(\\*\\), COUNT\\(DISTINCT client_uid\\)").
		WillReturnRows(sqlmock.NewRows([]string{
			"runs", "delvers", "floors_cleared", "deepest_floor",
			"gold_banked", "banked_runs", "failed_runs", "active_runs",
		}).AddRow(12, 5, 84, 37, 9000, 7, 5, 2))
	mock.ExpectQuery("SELECT COALESCE\\(tier, 'normal'\\), COUNT\\(\\*\\)").
		WillReturnRows(sqlmock.NewRows([]string{
			"tier", "runs", "floors_cleared", "deepest_floor",
		}).AddRow("hell", 3, 24, 20).
			AddRow("legacy", 1, 2, 2).
			AddRow("normal", 8, 58, 37))
	mock.ExpectCommit()
}
