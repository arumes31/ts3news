package bot

import (
	"strings"
	"testing"
)

func TestAbyssRunAwarenessContracts(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	for _, name := range []string{"abyssRunAwarenessBelt", "abyssRunAwarenessMiniHUD"} {
		var rendered strings.Builder
		if err := server.tmpl.ExecuteTemplate(&rendered, name, nil); err != nil {
			t.Fatalf("render %s: %v", name, err)
		}
	}
	script := server.tmpl.Lookup("abyssRunAwarenessJS")
	if script == nil {
		t.Fatal("Abyss run-awareness script partial is missing")
	}
	for _, required := range []string{
		"function updateProjectedHP",
		"function renderRunAwarenessConsumables",
		"function updateAbyssRunAwareness",
		"ab_run_started_at",
		"ab_run_floors",
		"abyssRunRecordBaseline",
		"replaceChildren",
		"threat estimate, not a guarantee",
		"bankLocked>0",
	} {
		if !strings.Contains(script.Tree.Root.String(), required) {
			t.Errorf("run-awareness script is missing %q", required)
		}
	}
}

func TestAbyssRunAwarenessAssetsAndHooks(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	longTerm, err := webAssets.ReadFile("webassets/abyss_longterm.html")
	if err != nil {
		t.Fatalf("read long-term module: %v", err)
	}
	pageSource := string(page) + string(longTerm)
	css, err := webAssets.ReadFile("webassets/abyss_run_awareness.css")
	if err != nil {
		t.Fatalf("read run-awareness stylesheet: %v", err)
	}
	for _, required := range []string{
		`{{asset "/static/abyss_run_awareness.css"}}`,
		`{{template "abyssRunAwarenessBelt" .}}`,
		`{{template "abyssRunAwarenessMiniHUD" .}}`,
		`{{template "abyssRunAwarenessJS" .}}`,
		"ab-float-delta",
		"function renderDepthRail",
		"' (+' + mb + '% STR)'",
		"mhLockWrap",
		"insuredTag",
		"buffStrip",
		"ab-pity-left",
		"ab-critical-hp",
		"hud.inert=!visible",
		"updateAbyssRunAwareness==='function'",
	} {
		if !strings.Contains(pageSource, required) {
			t.Errorf("Abyss page is missing run-awareness hook %q", required)
		}
	}
	for _, required := range []string{
		".ab-hpforecast",
		".ab-run-awareness-forecast",
		".ab-consumable-belt",
		".ab-mh-actions",
		".ab-mh-record.is-record",
		".abyss-stage.ab-critical-hp",
		"@media (prefers-reduced-motion: reduce)",
		"@media (forced-colors: active)",
	} {
		if !strings.Contains(string(css), required) {
			t.Errorf("run-awareness CSS is missing %q", required)
		}
	}
}
