package bot

import (
	"os"
	"strings"
	"testing"
)

func TestAbyssNavigationContracts(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	partial := server.tmpl.Lookup("abyss-navigation")
	if partial == nil {
		t.Fatal("Abyss navigation template is missing")
	}
	for _, required := range []string{
		"function buildAbyssTabs",
		"{key:'shop',label:'🜲 Shop'}",
		"{key:'forge',label:'⚒️ Forge'}",
		"panel.hidden=candidate.key!==group.key",
		"function activateDeepLink",
		"forge:'#abyssForgePanel'",
		"workshop:'#abyssWorkshop'",
		"/abyss/tree#talents",
		"function initCollapsiblePanels",
		"localStorage.setItem(key,open?'1':'0')",
		"abyssLibrarySearch",
		"row.dataset.navDisplay",
		"function initOverflow",
		"record.querySelector('#prestigePreview')",
		"function updateAbyssTitle",
		"Abyss · F",
		"+' escrow'",
		"function initShortcuts",
		"visibleButton('btnDescend')",
		"visibleButton('btnBank')",
		"document.getElementById('logSearch')",
		"event.key==='?'",
	} {
		if !strings.Contains(partial.Tree.Root.String(), required) {
			t.Errorf("navigation module is missing %q", required)
		}
	}
}

func TestAbyssNavigationAssetsAndIntegration(t *testing.T) {
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
	css, err := webAssets.ReadFile("webassets/abyss_navigation.css")
	if err != nil {
		t.Fatalf("read navigation CSS: %v", err)
	}
	for _, required := range []string{
		`{{asset "/static/abyss_navigation.css"}}`,
		`{{template "abyss-navigation" .}}`,
		`data-abyss-section="shop"`,
		`data-abyss-section="forge"`,
		"var lootFirst=inRun || abSettingGet",
		"row.classList.toggle('ab-loot-first'",
		"key:'ab_compact'",
	} {
		if !strings.Contains(pageSource, required) {
			t.Errorf("Abyss page is missing navigation hook %q", required)
		}
	}
	for _, required := range []string{
		".abyss-command-page .ab-tabs",
		"position: sticky",
		".ab-panel-collapsed",
		".ab-library-tools",
		"body.ab-compact",
		".ab-overflow",
		".ab-loot-first .abyss-side-right",
		".ab-shortcut-help",
		"@media (prefers-reduced-motion: reduce)",
		"@media (forced-colors: active)",
	} {
		if !strings.Contains(string(css), required) {
			t.Errorf("navigation CSS is missing %q", required)
		}
	}
	routes, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatalf("read web routes: %v", err)
	}
	if !strings.Contains(string(routes), `/static/abyss_navigation.css`) {
		t.Error("navigation stylesheet is not served")
	}
}
