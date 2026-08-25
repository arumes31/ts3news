package bot

import (
	"os"
	"strings"
	"testing"
)

func TestAbyssPlayerExperienceContracts(t *testing.T) {
	t.Parallel()
	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if server.tmpl.Lookup("abyssPlayerExperienceJS") == nil {
		t.Fatal("player experience script template is missing")
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	module, err := webAssets.ReadFile("webassets/abyss_player_experience.html")
	if err != nil {
		t.Fatal(err)
	}
	css, err := webAssets.ReadFile("webassets/abyss_player_experience.css")
	if err != nil {
		t.Fatal(err)
	}
	navigation, err := webAssets.ReadFile("webassets/abyss_navigation.html")
	if err != nil {
		t.Fatal(err)
	}
	source := string(page) + string(module) + string(css) + string(navigation)
	for _, required := range []string{
		"Abyss command palette", "Ctrl/⌘ K", "abCommandResults", "resumeRun",
		"abCurrentObjective", "abConnectionState", "abUTCClock", "abPageFreshness",
		"abCurrentSection", "abFilterSummary", "ab_player_scale", "ab_player_cv",
		"ab-cv-protan", "ab-cv-deutan", "ab-cv-tritan", "ab_player_motion",
		"ab_player_log_density", "ab-focus-mode", "ab-calm-mode", "exportSettings",
		"version:1", "Object.prototype.hasOwnProperty.call", "resetSettings",
		"Run report copied", "abyss-run-report.txt", "window.print", "updateTitleAlert",
		"slice(-12)", "textContent.trim()",
		"Copy support bundle", "Nothing is sent automatically", "supportBundleText",
		"excludes names, user IDs, combat logs, and random state",
		"{key:'shop',label:'🜲 Shop'}", "{key:'forge',label:'⚒️ Forge'}",
		`{{asset "/static/abyss_player_experience.css"}}`, `{{template "abyssPlayerExperienceJS" .}}`,
	} {
		if !strings.Contains(source, required) {
			t.Errorf("player experience layer is missing %q", required)
		}
	}
	for _, forbidden := range []string{"registerAbyssSafeRetry", "opts.retry", "ab-toast-retry", "retry your last action"} {
		if strings.Contains(string(module), forbidden) {
			t.Errorf("player experience layer contains forbidden retry mechanism %q", forbidden)
		}
	}
	webSource, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(webSource), "/static/abyss_player_experience.css") {
		t.Error("player experience stylesheet is not served")
	}
}

func TestAbyssPlayerExperienceImportIsDeviceLocalAndWhitelisted(t *testing.T) {
	t.Parallel()
	module, err := webAssets.ReadFile("webassets/abyss_player_experience.html")
	if err != nil {
		t.Fatal(err)
	}
	source := string(module)
	for _, required := range []string{"var keys=[", "parsed.version===1", "typeof values[key]==='string'", "values[key].length<=64", "localStorage.setItem"} {
		if !strings.Contains(source, required) {
			t.Errorf("portable settings guard is missing %q", required)
		}
	}
	start := strings.Index(source, "async function importSettings")
	if start < 0 {
		t.Fatal("display settings import function is missing")
	}
	end := strings.Index(source[start:], "async function resetSettings")
	if end < 0 || strings.Contains(source[start:start+end], "abPost(") {
		t.Error("display settings import must remain device-local")
	}
}

func TestAbyssSupportBundleRequiresConsentAndExcludesPrivateRunData(t *testing.T) {
	t.Parallel()
	module, err := webAssets.ReadFile("webassets/abyss_player_experience.html")
	if err != nil {
		t.Fatal(err)
	}
	source := string(module)
	start := strings.Index(source, "function supportBundleText")
	end := strings.Index(source, "async function copySupportBundle")
	if start < 0 || end <= start {
		t.Fatal("bounded support bundle functions are missing")
	}
	bundle := source[start:end]
	for _, forbidden := range []string{"session_id", "recent_logs", "random_seed", "owner_uid", ".name"} {
		if strings.Contains(bundle, forbidden) {
			t.Errorf("support bundle exposes private field %q", forbidden)
		}
	}
	consent := source[end:]
	for _, required := range []string{"await confirmModal", "Nothing is sent automatically", "navigator.clipboard.writeText"} {
		if !strings.Contains(consent, required) {
			t.Errorf("support bundle consent flow is missing %q", required)
		}
	}
}
