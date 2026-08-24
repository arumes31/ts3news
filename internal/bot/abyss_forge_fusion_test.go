package bot

import (
	"testing"

	"ts3news/internal/content"
)

func TestSelectFusionSurvivorAvoidsOwnedDuplicate(t *testing.T) {
	items := []content.Gear{
		{ID: "owned", Stats: content.Stats{STR: 100}},
		{ID: "fresh", Stats: content.Stats{STR: 80}},
	}
	got, err := selectFusionSurvivor(items, true, map[string]bool{"owned": true})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "fresh" {
		t.Fatalf("survivor = %q, want fresh", got.ID)
	}
	if _, err := selectFusionSurvivor(items[:1], true, map[string]bool{"owned": true}); err == nil {
		t.Fatal("expected duplicate-only fusion to be rejected")
	}
}
