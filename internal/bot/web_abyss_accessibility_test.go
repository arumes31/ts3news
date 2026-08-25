package bot

import (
	"os"
	"strings"
	"testing"
)

func TestAbyssAccessibilityContracts(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	partial := server.tmpl.Lookup("abyss-accessibility")
	if partial == nil {
		t.Fatal("Abyss accessibility template is missing")
	}
	source := partial.Tree.Root.String()
	for _, required := range []string{
		`id="abyssMobileActions"`,
		`data-mobile-action="btnDescend"`,
		`data-mobile-action="btnBank"`,
		`aria-controls="prepPanel"`,
		"function initTabSwipe",
		"Math.abs(dx)<70",
		"function initConsumableDialogFocus",
		"!card.contains(document.activeElement)",
		"function enhanceRarityGlyphs",
		"var rarityGlyphs=['●','■','▲','◆'",
		"function abyssHaptic",
		"localStorage.getItem(key)",
		"'(pointer: coarse)'",
		"navigator.vibrate(pattern)",
		"preference('ab_haptic','0')==='1'",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("accessibility module is missing %q", required)
		}
	}
}

func TestAbyssAccessibilityAssetsAndSettings(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	pageSource := string(page)
	for _, required := range []string{
		`{{asset "/static/abyss_accessibility.css"}}`,
		`{{template "abyss-accessibility" .}}`,
		`id="abyssBanner" role="status" aria-live="polite" aria-atomic="true"`,
		`id="abToastHost" role="status" aria-live="polite" aria-relevant="additions text"`,
		`key:'ab_contrast'`,
		`key:'ab_logsize'`,
		`key:'ab_haptic'`,
		`def:false`,
	} {
		if !strings.Contains(pageSource, required) {
			t.Errorf("Abyss page is missing accessibility hook %q", required)
		}
	}

	css, err := webAssets.ReadFile("webassets/abyss_accessibility.css")
	if err != nil {
		t.Fatalf("read accessibility CSS: %v", err)
	}
	cssSource := string(css)
	for _, required := range []string{
		".ab-mobile-actions",
		"min-height: 48px",
		"@media (pointer: coarse)",
		"min-height: 44px",
		"body.ab-high-contrast",
		`body[data-ab-log-size="s"]`,
		`body[data-ab-log-size="m"]`,
		`body[data-ab-log-size="l"]`,
		".ab-rarity-glyph",
		"@media (prefers-reduced-motion: reduce)",
		".ab-coin-arc",
		"@media (forced-colors: active)",
	} {
		if !strings.Contains(cssSource, required) {
			t.Errorf("accessibility CSS is missing %q", required)
		}
	}

	routes, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatalf("read web routes: %v", err)
	}
	if !strings.Contains(string(routes), `/static/abyss_accessibility.css`) {
		t.Error("accessibility stylesheet is not served")
	}
}

func TestSharedModalKeyboardContracts(t *testing.T) {
	t.Parallel()

	partial, err := webAssets.ReadFile("webassets/partials.html")
	if err != nil {
		t.Fatalf("read shared partials: %v", err)
	}
	source := string(partial)
	for _, required := range []string{
		`id="sharedModal" class="modal-overlay" aria-hidden="true"`,
		`setAttribute('aria-hidden', 'false')`,
		`setAttribute('aria-hidden', 'true')`,
		"function trapSharedModalFocus",
		"!card.contains(document.activeElement)",
		"e.key === 'Escape'",
		"modalReturnFocus.focus()",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("shared modal is missing keyboard contract %q", required)
		}
	}
}
