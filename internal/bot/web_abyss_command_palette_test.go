package bot

import (
	"os"
	"strings"
	"testing"
)

func TestAbyssCommandPaletteContracts(t *testing.T) {
	t.Parallel()
	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if server.tmpl.Lookup("abyssCommandPaletteJS") == nil {
		t.Fatal("command palette script template is missing")
	}
	module, err := webAssets.ReadFile("webassets/abyss_command_palette.html")
	if err != nil {
		t.Fatal(err)
	}
	css, err := webAssets.ReadFile("webassets/abyss_command_palette.css")
	if err != nil {
		t.Fatal(err)
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	source := string(module) + string(css) + string(page)
	for _, required := range []string{
		`role="combobox"`, `aria-autocomplete="list"`, `aria-controls="abCommandResults"`,
		`aria-activedescendant`, `role="listbox"`, `setAttribute('role','option')`,
		`setAttribute('aria-disabled'`, `setAttribute('aria-selected'`, `role="status"`,
		`event.key==='Home'`, `event.key==='End'`, `event.key==='/'`, `isTypingTarget`,
		`ab_command_recent_v2`, `maxRecent=8`, `JSON.stringify(recent.slice(0,maxRecent))`,
		`subsequenceScore`, `button.dataset.commandId`, `commandDisabledReason`,
		`document.createTextNode`, `appendHighlighted`, `replaceChildren`, `removeAttribute('aria-activedescendant')`,
		`[data-abyss-section] button[id]`, `.ab-tab[data-tab-key]`,
		`{{asset "/static/abyss_command_palette.css"}}`, `{{template "abyssCommandPaletteJS" .}}`,
	} {
		if !strings.Contains(source, required) {
			t.Errorf("command palette is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"label.innerHTML", "button.innerHTML", "host.innerHTML", "insertAdjacentHTML(item",
	} {
		if strings.Contains(string(module), forbidden) {
			t.Errorf("command palette uses unsafe result rendering %q", forbidden)
		}
	}
	webSource, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(webSource), "/static/abyss_command_palette.css") {
		t.Error("command palette stylesheet is not served")
	}
}

func TestAbyssCommandPaletteKeepsDisabledActionsDiscoverable(t *testing.T) {
	t.Parallel()
	module, err := webAssets.ReadFile("webassets/abyss_command_palette.html")
	if err != nil {
		t.Fatal(err)
	}
	source := string(module)
	for _, required := range []string{
		"function unavailableReason", "Resolve the revival decision first.",
		"Unavailable while live combat is resolving.", "if(!item.enabled)",
		"item.label+' — '+reason", "input.focus()", "return;",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("disabled command explanation is missing %q", required)
		}
	}
	if strings.Contains(source, "if(!button.id||button.disabled") {
		t.Error("disabled controls must remain discoverable in the palette")
	}
}
