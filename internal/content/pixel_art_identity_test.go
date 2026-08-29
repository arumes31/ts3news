package content

import (
	"testing"
)

func TestEveryCatalogEntryHasUniquePixelArtIdentity(t *testing.T) {
	entries := PixelArtCatalog()
	if len(entries) < 2700 {
		t.Fatalf("exact icon catalog unexpectedly small: %d entries", len(entries))
	}
	keys := make(map[string]struct{}, len(entries))
	coordinates := make(map[string]string, len(entries))
	kinds := make(map[string]int)
	for _, entry := range entries {
		if _, exists := keys[entry.Key]; exists {
			t.Fatalf("duplicate exact icon key %q", entry.Key)
		}
		keys[entry.Key] = struct{}{}
		coordinate := entry.Asset + ":" + string(rune(entry.Column)) + ":" + string(rune(entry.Row))
		if previous, exists := coordinates[coordinate]; exists {
			t.Fatalf("exact icon coordinate collision: %q and %q", previous, entry.Key)
		}
		coordinates[coordinate] = entry.Key
		kinds[entry.Kind]++
		resolved, ok := PixelArtByKey(entry.Key)
		if !ok || resolved != entry {
			t.Fatalf("exact icon lookup failed for %q", entry.Key)
		}
	}
	for _, kind := range []string{"gear", "consumable", "artifact", "skill", "ultimate", "monster", "pet"} {
		if kinds[kind] == 0 {
			t.Fatalf("exact icon catalog has no %s entries", kind)
		}
	}
	t.Logf("verified %d collision-free exact icons: %#v", len(entries), kinds)
}

func TestMobCatalogInitializationIsIdempotent(t *testing.T) {
	initMobs()
	before := len(baseMobs)
	for range 10 {
		initMobs()
	}
	if len(baseMobs) != before {
		t.Fatalf("mob catalog grew from %d to %d after repeated initialization", before, len(baseMobs))
	}
}
