package bot

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestAbyssFeedbackContracts(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	partial := server.tmpl.Lookup("abyss-feedback")
	if partial == nil {
		t.Fatal("Abyss feedback template is missing")
	}
	source := partial.Tree.Root.String()
	for _, required := range []string{
		"function humanError",
		"Couldn't save a confirmed result",
		"function enhanceNetworkBanner",
		"method:'HEAD'",
		"Check now",
		"function showSessionExpiry",
		"abyss_session_resume_v1",
		"sessionStorage.setItem",
		"Session expired",
		"location.assign('/denied')",
		"function restoreSessionView",
		"visibilitychange",
		"10*60*1000",
		"Resume your run at floor",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("feedback module is missing %q", required)
		}
	}
}

func TestAbyssFeedbackPageIntegration(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	source := string(page)
	for _, required := range []string{
		`{{asset "/static/abyss_feedback.css"}}`,
		`{{template "abyss-feedback" .}}`,
		"kids.length > 3",
		"!kids[i].classList.contains('bad')",
		"setTimeout(function(){lastActionBtnTimer=null;",
		"},400)",
		"r.status===401",
		"abyssLootSettingsRevision",
		"revision!==abyssLootSettingsRevision",
		"optimistic=!previous",
		"button.disabled=true",
		"Insurance returns",
		"netTimer = setInterval",
		"method: 'HEAD'",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("Abyss page is missing feedback contract %q", required)
		}
	}

	css, err := webAssets.ReadFile("webassets/abyss_feedback.css")
	if err != nil {
		t.Fatalf("read feedback CSS: %v", err)
	}
	for _, required := range []string{".ab-network-check", "min-height: 44px", "prefers-reduced-motion", "forced-colors"} {
		if !strings.Contains(string(css), required) {
			t.Errorf("feedback CSS is missing %q", required)
		}
	}

	routes, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatalf("read web routes: %v", err)
	}
	if strings.Count(string(routes), `/static/abyss_feedback.css`) != 1 {
		t.Error("feedback stylesheet must have exactly one route")
	}
}

func TestAuthAPIReportsUnauthorizedStatus(t *testing.T) {
	t.Parallel()

	server := &WebServer{}
	called := false
	handler := server.authAPI(func(http.ResponseWriter, *http.Request, string) { called = true })
	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest(http.MethodPost, "/api/abyss/descend", nil))
	if called {
		t.Fatal("protected handler ran without a session")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(recorder.Body.String(), `"error":"unauthenticated"`) {
		t.Fatalf("body = %q, want unauthenticated JSON", recorder.Body.String())
	}
}

func TestAbyssFeedbackNeverReplaysMutations(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	partial, err := webAssets.ReadFile("webassets/abyss_feedback.html")
	if err != nil {
		t.Fatalf("read feedback module: %v", err)
	}
	combined := string(page) + string(partial)
	for _, forbidden := range []string{"registerAbyssSafeRetry", "clearAbyssSafeRetry", "opts.retry", "ab-toast-retry"} {
		if strings.Contains(combined, forbidden) {
			t.Errorf("feedback must not replay ambiguous mutations: found %q", forbidden)
		}
	}
}
