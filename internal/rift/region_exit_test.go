package rift

import (
	"encoding/json"
	"testing"
)

func TestRegionalExitDestinations(t *testing.T) {
	seen := map[[2]float64]bool{}
	for i, level := range Campaign() {
		for room, arena := range level.Rooms {
			e := arena.Exit
			if e == nil {
				t.Fatal("missing regional exit")
			}
			for _, obstacle := range arena.solidObstacles() {
				if contains(obstacle, e.X, e.Y, 15) {
					t.Fatal("blocked exit")
				}
			}
			for _, hazard := range arena.Hazards {
				if contains(hazard.Obstacle, e.X, e.Y, 14) {
					t.Fatal("unsafe exit")
				}
			}
			if arena.Objective == "escape_collapse" || arena.Objective == "carry_relic" || arena.Objective == "escort_spirit" {
				r := testRun()
				r.Level = &level
				r.Room = room
				r.beginRoomObjective()
				if r.RoomObjective.Zone.X != e.X || r.RoomObjective.Zone.Y != e.Y {
					t.Fatal("objective ignores exit")
				}
			}
		}
		if i%10 == 0 {
			e := level.Rooms[0].Exit
			seen[[2]float64{e.X, e.Y}] = true
		}
	}
	if len(seen) != 10 {
		t.Fatal("regions share exits")
	}
}

func TestExitCopyAndLegacyDestinations(t *testing.T) {
	levels := Campaign()
	levels[0].Rooms[0].Exit.X = 99
	if Campaign()[0].Rooms[0].Exit.X == 99 {
		t.Fatal("exit pointer shared")
	}
	r := testRun()
	level := Campaign()[0]
	r.Level = &level
	r.Room = 0
	r.Level.Rooms[0].Exit = nil
	r.beginRelicObjective()
	if r.RoomObjective.Zone.X != 1450 || r.RoomObjective.Zone.Y != 480 {
		t.Fatal("legacy relic exit changed")
	}
	r.Level.Rooms[0].Exit = &ArenaEntrance{1500, 320}
	r.beginRelicObjective()
	if r.RoomObjective.Zone.X != 1500 || r.RoomObjective.Zone.Y != 320 {
		t.Fatal("frozen exit ignored")
	}
}

func TestExitSurvivesSaveAndCompletesCollapse(t *testing.T) {
	r := collapseTestRun()
	r.Level.Rooms[r.Room].Exit = &ArenaEntrance{1510, 320}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	saved.beginCollapseObjective()
	saved.Player.X, saved.Player.Y = 1510, 320
	saved.tick(Input{}, .02)
	if !saved.RoomObjective.Complete || saved.Status != "cleared" {
		t.Fatal("frozen destination did not complete collapse")
	}
}
