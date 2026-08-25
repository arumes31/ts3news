package bot

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestAbyssTrapPassChance(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		dodge, depth, want int
	}{
		{0, 1, 49},
		{20, 30, 40},
		{100, 20, 95},
		{0, 100, 10},
	} {
		if got := abyssTrapPassChance(tc.dodge, tc.depth); got != tc.want {
			t.Errorf("abyssTrapPassChance(%d, %d) = %d, want %d", tc.dodge, tc.depth, got, tc.want)
		}
	}
}

func TestPrepareAbyssTraversalEvents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		typ   string
		check func(t *testing.T, event abyssTraversalEvent)
	}{
		{"cursed_elevator", func(t *testing.T, event abyssTraversalEvent) {
			if len(event.Destinations) != 2 || event.Destinations[0].Depth != 22 || event.Destinations[1].Depth != 24 {
				t.Fatalf("elevator destinations = %#v", event.Destinations)
			}
		}},
		{"graveyard", func(t *testing.T, event abyssTraversalEvent) {
			if event.GhostName == "" || event.DeathDepth <= 0 || event.DeathDepth >= 20 {
				t.Fatalf("graveyard identity = %#v", event)
			}
		}},
		{"unstable_portal", func(t *testing.T, event abyssTraversalEvent) {
			if len(event.MissedFloors) != 3 || strings.Join(event.MissedFloors, " · ") != "Combat · Rest · Event" {
				t.Fatalf("sealed missed floors = %#v", event.MissedFloors)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.typ, func(t *testing.T) {
			state := map[string]any{"type": tc.typ, "depth": 20}
			prepareAbyssTraversalEvent(state, 20)
			raw, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			var event abyssTraversalEvent
			if err := json.Unmarshal(raw, &event); err != nil {
				t.Fatal(err)
			}
			tc.check(t, event)
		})
	}
}

func TestAbyssTraversalRoomUIContracts(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	server, err := os.ReadFile("web_abyss_traversal_rooms.go")
	if err != nil {
		t.Fatal(err)
	}
	joined := string(page) + string(server)
	for _, contract := range []string{
		"elevator_choose_0", "destination.label", "pass_chance", "trap_attempt",
		"portal_enter", "What you missed", "ghost_name", "death_depth", "graveyard_honor",
	} {
		if !strings.Contains(joined, contract) {
			t.Errorf("traversal-room contract is missing %q", contract)
		}
	}
}
