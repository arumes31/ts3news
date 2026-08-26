package bot

import (
	"strings"
	"testing"
	"time"

	"ts3news/internal/content"
)

func TestStampAbyssGearProvenanceSetsMissingReceiptFields(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 26, 12, 34, 56, 0, time.FixedZone("test", 2*60*60))
	gear := content.Gear{}
	stampAbyssGearProvenance(&gear, 25, " Gorgoroth the Firelord ", now)

	if gear.FoundAt != "2026-08-26T10:34:56Z" || gear.FoundDepth != 25 || gear.FoundBoss != "Gorgoroth the Firelord" {
		t.Fatalf("stamped provenance = %#v", gear)
	}

	stampAbyssGearProvenance(&gear, 99, "Replacement", now.Add(time.Hour))
	if gear.FoundAt != "2026-08-26T10:34:56Z" || gear.FoundDepth != 25 || gear.FoundBoss != "Gorgoroth the Firelord" {
		t.Fatalf("existing provenance was overwritten: %#v", gear)
	}
}

func TestGearProvenanceIsBoundedAndRedactsUnidentifiedGear(t *testing.T) {
	t.Parallel()

	gear := content.Gear{
		FoundAt:    "2026-08-26T10:34:56Z",
		FoundDepth: 25,
		FoundBoss:  strings.Repeat("B", 90),
	}
	got := gearProvenance(gear)
	if !strings.Contains(got, "Abyss depth 25") || !strings.Contains(got, "2026-08-26 UTC") || !strings.Contains(got, "…") {
		t.Fatalf("bounded provenance = %q", got)
	}

	gear.Unidentified = true
	if got := gearProvenance(gear); got != "" {
		t.Fatalf("unidentified provenance leaked as %q", got)
	}
}

func TestAbyssProvenancePresentationContracts(t *testing.T) {
	t.Parallel()

	for name, tokens := range map[string][]string{
		"webassets/abyss.html":              {".Provenance", "data-tip="},
		"webassets/armory.html":             {`title="{{.Provenance}}"`, `class="gear-provenance"`},
		"webassets/inventory.html":          {`title="{{.Provenance}}"`, `class="gear-provenance"`},
		"webassets/abyss_inventory_ui.html": {"item.provenance", "data-tip"},
	} {
		body, err := webAssets.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, token := range tokens {
			if !strings.Contains(string(body), token) {
				t.Errorf("%s missing provenance contract %q", name, token)
			}
		}
	}
}
