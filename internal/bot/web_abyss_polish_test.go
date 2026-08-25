package bot

import (
	"os"
	"strings"
	"testing"
)

func TestAbyssPolishContracts(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	partial := server.tmpl.Lookup("abyss-polish")
	if partial == nil {
		t.Fatal("Abyss polish template is missing")
	}
	source := partial.Tree.Root.String()
	for _, required := range []string{
		"function recapCanvas",
		"function recapRows",
		"['Gold banked'",
		"['Best drop'",
		"['Deaths'",
		"canvas.toBlob",
		"new ClipboardItem",
		"abyss-run-recap.png",
		"className='ghost ab-share-recap'",
		"wrapAfter('showRunSummary'",
		"wrapAfter('showBankSummary'",
		"function renderPersonalDashboard",
		"Average depth",
		"Gold banked by day",
		"function causeBars",
		"function depthBand",
		"document.body.dataset.abDepthBand",
		"document.body.dataset.abSeason",
		"function celebrate",
		"for(var i=0;i<24;i++)",
		"ab_first_boss_confetti",
		"reduceMotion",
		"key:'ab_sound'",
		"readLiveCombatPreference('abyssCombatAudio','off')",
		"playLiveCombatCue('ready'",
		"playLiveCombatCue('defeat'",
		"if(rarity)playLiveCombatCue('cast'",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("polish module is missing %q", required)
		}
	}
}

func TestAbyssPolishAssetsAndIntegration(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	source := string(page)
	for _, required := range []string{
		`{{asset "/static/abyss_polish.css"}}`,
		`{{template "abyss-polish" .}}`,
		`data-depth="{{.Depth}}"`,
		`data-gold="{{.Gold}}"`,
		`data-victory="{{.Victory}}"`,
		"recordSessionRun(false, curDepth, 0, 'Conceded')",
		"localStorage.getItem('ab_run_causes')",
		"causes.slice(0,50)",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("Abyss page is missing polish contract %q", required)
		}
	}
	if !strings.Contains(source, "var finishedDepth=curDepth") ||
		!strings.Contains(source, "recordSessionRun(false, finishedDepth, 0, 'Failed revival')") ||
		!strings.Contains(source, "['Deepest floor', finishedDepth]") {
		t.Error("failed-revival summary must preserve the completed depth before resetting run state")
	}

	css, err := webAssets.ReadFile("webassets/abyss_polish.css")
	if err != nil {
		t.Fatalf("read polish CSS: %v", err)
	}
	cssSource := string(css)
	for _, required := range []string{
		`body[data-ab-depth-band="abyssal"]`,
		`body[data-ab-season="winter"]`,
		"--ab-season-accent",
		".ab-personal-dashboard",
		".ab-personal-charts",
		".ab-cause-row",
		".ab-confetti",
		"@keyframes ab-confetti-fall",
		"@media (prefers-reduced-motion: reduce)",
		"@media (forced-colors: active)",
	} {
		if !strings.Contains(cssSource, required) {
			t.Errorf("polish CSS is missing %q", required)
		}
	}

	routes, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatalf("read web routes: %v", err)
	}
	if strings.Count(string(routes), `/static/abyss_polish.css`) != 1 {
		t.Error("polish stylesheet must have exactly one route")
	}
}

func TestAbyssSoundRemainsOptIn(t *testing.T) {
	t.Parallel()

	feedback, err := webAssets.ReadFile("webassets/abyss_combat_feedback.html")
	if err != nil {
		t.Fatalf("read combat feedback: %v", err)
	}
	source := string(feedback)
	if !strings.Contains(source, "readLiveCombatPreference('abyssCombatAudio','off')==='on'") {
		t.Error("combat audio must remain muted by default")
	}
	if !strings.Contains(source, "if(!liveCombatAudioEnabled)return null") {
		t.Error("audio context must not start before opt-in")
	}
}
