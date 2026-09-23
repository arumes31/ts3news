package rift

import (
	"encoding/json"
	"testing"
)

func TestCheckerboardHazardsAlwaysLeaveSafeTiles(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		if level.ID%10 != 9 {
			continue
		}
		count++
		hazards := level.Rooms[2].Hazards
		if len(hazards) != 8 {
			t.Fatalf("mission %d missing eight checkerboard tiles", level.ID)
		}
		raw, err := json.Marshal(hazards)
		if err != nil {
			t.Fatal(err)
		}
		var saved []Hazard
		if err = json.Unmarshal(raw, &saved); err != nil {
			t.Fatal(err)
		}
		for _, clock := range []float64{0, 1.2, 1.3, 1.7, 2.21, 3, 4.2, 4.3, 4.7, 5.21, 6, 7.3, 10.3} {
			active := 0
			for _, h := range saved {
				phase := h.Phase(clock)
				if phase >= 1.2 && phase < 1.2+h.Duration {
					active++
				}
			}
			if (clock == 1.3 || clock == 4.3) && active != 4 {
				t.Fatal("checkerboard group did not activate")
			}
			if active > 4 {
				t.Fatalf("mission %d has %d active tiles", level.ID, active)
			}
		}
		for i, h := range saved {
			if !h.Tile || !h.Jumpable || h.Period != 6 || h.Duration != 1 || h.Offset != float64((i/4+i%4)%2)*3 {
				t.Fatal("invalid alternating tile")
			}
			if h.Phase(1.3) >= 1.2 && h.Phase(1.3) < 1.2+h.Duration {
				if h.Phase(4.3) >= 1.2 && h.Phase(4.3) < 1.2+h.Duration {
					t.Fatalf("tile %d never alternated", i)
				}
			}
		}
	}
	if count != 10 {
		t.Fatal("missing checkerboard rooms")
	}
}
