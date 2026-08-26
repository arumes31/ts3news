package bot

import (
	"strings"
	"testing"
)

func TestAbyssLootSignalsStayInDomainTemplate(t *testing.T) {
	t.Parallel()

	pageBytes, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	partialBytes, err := webAssets.ReadFile("webassets/abyss_loot_signals.html")
	if err != nil {
		t.Fatal(err)
	}
	page, partial := string(pageBytes), string(partialBytes)
	if !strings.Contains(page, `template "abyssLootSignals" .`) {
		t.Fatal("Abyss page does not load the loot-signals domain template")
	}
	for _, contract := range []string{
		`define "abyssLootSignals"`, `id="bountyCard"`, `id="pityCount"`,
		`id="pityFill"`, `id="dropStreakCount"`, `id="dropStreakFill"`,
		`template "abyss-featured-drops" .FeaturedDrops`,
	} {
		if !strings.Contains(partial, contract) {
			t.Errorf("loot-signals template missing contract %q", contract)
		}
	}
	for _, leaked := range []string{`id="bountyCard"`, `id="pityFill"`, `id="dropStreakFill"`} {
		if strings.Contains(page, leaked) {
			t.Errorf("domain markup %q drifted back into abyss.html", leaked)
		}
	}
}
