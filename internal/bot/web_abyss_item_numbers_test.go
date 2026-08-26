package bot

import (
	"strings"
	"testing"
)

func TestAbyssItemNumbersRemainPresentationOnly(t *testing.T) {
	t.Parallel()

	assets := make(map[string]string)
	for _, name := range []string{
		"webassets/abyss_item_numbers.html",
		"webassets/abyss_longterm.html",
		"webassets/abyss_player_experience.html",
		"webassets/abyss.html",
		"webassets/armory.html",
		"webassets/inventory.html",
		"webassets/abyss_forge_planner.html",
	} {
		body, err := webAssets.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		assets[name] = string(body)
	}

	module := assets["webassets/abyss_item_numbers.html"]
	for _, contract := range []string{
		`define "abyssItemNumbersJS"`, `ab_item_numbers`, `notation:'compact'`,
		`data-item-number`, `Exact value: `, `aria-label`, `window.AbyssItemNumbers`,
	} {
		if !strings.Contains(module, contract) {
			t.Errorf("item-number module missing contract %q", contract)
		}
	}
	for _, page := range []string{"webassets/abyss.html", "webassets/armory.html", "webassets/inventory.html"} {
		if !strings.Contains(assets[page], `template "abyssItemNumbersJS"`) {
			t.Errorf("%s does not load the shared item-number renderer", page)
		}
	}
	if !strings.Contains(assets["webassets/abyss_longterm.html"], `label:'Item stat numbers'`) {
		t.Error("Abyss settings do not expose the exact/compact item-number preference")
	}
	if !strings.Contains(assets["webassets/abyss_player_experience.html"], `'ab_item_numbers'`) {
		t.Error("portable display settings do not include item-number mode")
	}
	for _, forbidden := range []string{"fetch(", "abPost(", "/api/"} {
		if strings.Contains(module, forbidden) {
			t.Errorf("presentation-only number module unexpectedly contains %q", forbidden)
		}
	}
}
