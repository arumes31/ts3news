package bot

import (
	"os"
	"strings"
	"testing"
)

func TestAbyssEventTypeLabels(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		`{"type":"merchant"}`:          "Abyssal Market",
		`{"type":"collapsed_passage"}`: "Collapsed Passage",
		`{"type":"abyssal_garden"}`:    "Abyssal Garden",
		`{"type":"sigil_chain"}`:       "Triune Sigil Hunt",
		`{"type":"unknown"}`:           "Unknown anomaly",
		`not-json`:                     "Unknown anomaly",
	}
	for raw, want := range tests {
		if got := abyssEventTypeLabel(raw); got != want {
			t.Errorf("abyssEventTypeLabel(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestAbyssMapTableUpgradeContract(t *testing.T) {
	t.Parallel()

	for _, upgrade := range sanctuaryUpgrades {
		if upgrade.Key == "map" && upgrade.Max == 1 && upgrade.CostTk == 50 {
			return
		}
	}
	t.Fatal("single-level 50-token Map Table upgrade is missing")
}

func TestAbyssFloorEventIntelligenceContracts(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	rooms, err := os.ReadFile("web_abyss_rooms.go")
	if err != nil {
		t.Fatal(err)
	}
	server, err := os.ReadFile("web_abyss.go")
	if err != nil {
		t.Fatal(err)
	}
	joined := string(page) + string(rooms) + string(server)
	for _, contract := range []string{
		"ab-sigil-ribbon", "ab-map-forecast", "takeAbyssEventPreview",
		"loot_hint", "abyssLootHint", "collapsed_passage", "passage_detour",
		"abyssal_garden", "garden_harvest", "abyssExplorerKeepsakeDue",
	} {
		if !strings.Contains(joined, contract) {
			t.Errorf("floor/event intelligence is missing %q", contract)
		}
	}
}
