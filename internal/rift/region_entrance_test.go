package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRegionsHaveDistinctSafeEntrances(t *testing.T) {
	seen := map[[2]float64]bool{}
	for region := 0; region < 10; region++ {
		var position [2]float64
		for layout := 0; layout < 10; layout++ {
			r := NewRunAtLevel("entry", testRun().Build, time.Unix(100, 0), nil, region*10+layout+1)
			for room := range r.Level.Rooms {
				r.Room = room
				r.spawnRoom()
				p := r.Player
				if p.X < 35 || p.X > 1565 || p.Y < 315 || p.Y > 490 {
					t.Fatal("entrance outside arena")
				}
				for _, o := range r.Arena().solidObstacles() {
					if contains(o, p.X, p.Y, actorClearance(&p)) {
						t.Fatal("entrance intersects wall")
					}
				}
				for _, h := range r.Arena().Hazards {
					if contains(h.Obstacle, p.X, p.Y, actorClearance(&p)) {
						t.Fatal("entrance intersects hazard envelope")
					}
				}
				if layout == 0 && room == 0 {
					position = [2]float64{p.X, p.Y}
				} else if position != [2]float64{p.X, p.Y} {
					t.Fatal("region arrival position changed between tiers")
				}
			}
		}
		if seen[position] {
			t.Fatalf("region %d reused another region's entry %v", region, position)
		}
		seen[position] = true
	}
}

func TestEntranceSaveDoesNotTeleportResumingPlayer(t *testing.T) {
	r := NewRunAtLevel("entry", testRun().Build, time.Unix(100, 0), nil, 31)
	r.Player.X = 710
	r.Player.Y = 420
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Player.X != 710 || saved.Player.Y != 420 {
		t.Fatal("resume teleported to entrance")
	}
	// A legacy frozen arena without the optional entrance retains the old start.
	saved.Level.Rooms[0] = Arena{}
	if err = json.Unmarshal([]byte(`{"name":"Legacy","hazards":[],"obstacles":[]}`), &saved.Level.Rooms[0]); err != nil {
		t.Fatal(err)
	}
	saved.spawnRoom()
	if saved.Player.X != 160 || saved.Player.Y != 410 {
		t.Fatal("legacy entrance changed")
	}
}

func TestEntranceCopiesAndFrozenTierTransition(t *testing.T) {
	levels := Campaign()
	levels[30].Rooms[1].Entrance.X = 90
	if Campaign()[30].Rooms[1].Entrance.X == 90 {
		t.Fatal("shared campaign entrance")
	}
	r := NewRunAtLevel("entry", testRun().Build, time.Unix(100, 0), nil, 31)
	r.Level.Rooms[1].Entrance = &ArenaEntrance{90, 450}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	saved.Room = 1
	saved.spawnRoom()
	if saved.Player.X != 90 || saved.Player.Y != 450 {
		t.Fatal("tier transition ignored frozen entrance")
	}
}
