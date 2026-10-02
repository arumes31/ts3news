package rift

import (
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestMissionStartEmitsOnlySelectedArenaEntry(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	now := time.Unix(100, 0)
	for id := 1; id <= LevelCount; id++ {
		run := NewRunAtLevel("single-plan", Build{Name: "Hero", Class: "vanguard", HP: 200}, now, catalog, id)
		counts := map[string]int{}
		for _, event := range run.Events {
			counts[event.Kind]++
			if (event.Kind == "area" || event.Kind == "arrival") && (event.X != run.Player.X || event.Y != run.Player.Y || event.Elevation != run.Player.Elevation) {
				t.Fatalf("mission %d entry cue has wrong placement", id)
			}
		}
		if counts["area"] != 1 || counts["arrival"] != 1 {
			t.Fatalf("mission %d entry events: %v", id, counts)
		}
		if run.Level.ID != id || run.Player.Name != "Hero" || run.Player.HP != 200 || run.LastMS != now.UnixMilli() || run.SavedAtMS != now.UnixMilli() || len(run.EncounterPlan) != 3 {
			t.Fatalf("mission %d lost initialization", id)
		}
	}
}
