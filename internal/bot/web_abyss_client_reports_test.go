package bot

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAbyssClientReportStoreSanitizesAggregatesAndRateLimits(t *testing.T) {
	t.Parallel()

	var store abyssClientReportStore
	now := time.Date(2026, time.August, 25, 1, 2, 3, 0, time.UTC)
	report := abyssClientErrorReport{
		Kind:   "script_error",
		Source: "https://example.test/abyss.js",
		Path:   "not-a-path",
		Line:   -4,
		Column: 11,
	}

	first, accepted := store.record("player-1", report, now)
	if !accepted {
		t.Fatal("first report was rate limited")
	}
	if first.Kind != "script_error" || first.Source != "other" || first.Path != "/abyss" {
		t.Fatalf("sanitized report = %+v", first)
	}
	if _, accepted := store.record("player-1", report, now.Add(time.Second)); accepted {
		t.Fatal("second report inside the rate window was accepted")
	}
	third, accepted := store.record("player-1", report, now.Add(3*time.Second))
	if !accepted || third.Count != 2 {
		t.Fatalf("aggregated report = %+v, accepted=%v", third, accepted)
	}

	snapshot := store.snapshot()
	if snapshot["received"] != int64(3) || snapshot["dropped"] != int64(1) {
		t.Fatalf("snapshot counters = %#v", snapshot)
	}
}

func TestAbyssClientReportStoreBoundsErrorGroups(t *testing.T) {
	t.Parallel()

	var store abyssClientReportStore
	now := time.Date(2026, time.August, 25, 1, 2, 3, 0, time.UTC)
	for index := range abyssClientReportMaxGroups + 20 {
		report := abyssClientErrorReport{
			Kind: "script_error", Source: "/static/abyss.js", Path: "/abyss", Line: index + 1,
		}
		if _, accepted := store.record(string(rune(index+1)), report, now.Add(time.Duration(index)*time.Second)); !accepted {
			t.Fatalf("unique report %d was unexpectedly rejected", index)
		}
	}
	store.mu.Lock()
	groups := len(store.reports)
	store.mu.Unlock()
	if groups != abyssClientReportMaxGroups {
		t.Fatalf("stored groups = %d, want %d", groups, abyssClientReportMaxGroups)
	}
	if top := store.snapshot()["top"].([]abyssClientErrorSummary); len(top) != 20 {
		t.Fatalf("operator snapshot contains %d groups, want 20", len(top))
	}
}

func TestHandleAbyssClientErrorAcceptsBoundedJSON(t *testing.T) {
	t.Parallel()

	server := &WebServer{}
	payload, err := json.Marshal(abyssClientErrorReport{
		Kind:   "script_error",
		Source: "/static/abyss.js",
		Path:   "/abyss",
		Line:   42,
	})
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/client-error", bytes.NewReader(payload))
	recorder := httptest.NewRecorder()
	server.handleAbyssClientError(recorder, request, "player-1")
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want JSON", contentType)
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["ok"] != true || response["accepted"] != true {
		t.Fatalf("response = %#v", response)
	}
}

func TestHandleAbyssClientErrorRejectsMultipleJSONValues(t *testing.T) {
	t.Parallel()

	server := &WebServer{}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/abyss/client-error",
		strings.NewReader(`{"kind":"script_error"} {"kind":"resource_error"}`),
	)
	recorder := httptest.NewRecorder()
	server.handleAbyssClientError(recorder, request, "player-1")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandleAbyssClientErrorRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	server := &WebServer{}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/abyss/client-error",
		strings.NewReader(`{"kind":"script_error","secret":"do not store"}`),
	)
	recorder := httptest.NewRecorder()
	server.handleAbyssClientError(recorder, request, "player-1")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}
