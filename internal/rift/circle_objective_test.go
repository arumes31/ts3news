package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func circleTestRun() *Run {
	r := NewRunAtLevel("circle", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
	r.Level.Rooms[0].Objective = "hold_circle"
	r.Level.Rooms[0].Hazards = nil
	r.spawnRoom()
	r.Enemies = []Actor{{ID: "far", Kind: "goblin", X: 1400, Y: 490, HP: 100, MaxHP: 100, Knockdown: 100}}
	return r
}
func TestCircleRequiresGroundedUncontestedPresence(t *testing.T) {
	for _, mode := range []string{"inside", "outside", "contested", "airborne", "paused", "dead"} {
		t.Run(mode, func(t *testing.T) {
			r := circleTestRun()
			if r.RoomObjective == nil {
				t.Fatal("circle missing")
			}
			zone := r.RoomObjective.Zone
			r.Player.X = zone.X
			r.Player.Y = zone.Y
			switch mode {
			case "outside":
				r.Player.X += zone.RadiusX + 1
			case "contested":
				r.Enemies[0].X = zone.X
				r.Enemies[0].Y = zone.Y
			case "airborne":
				r.Player.Jump = .5
			case "paused":
				r.SetPaused(true, time.Unix(100, 0))
			case "dead":
				r.Player.HP = 0
			}
			r.Step(Input{}, time.Unix(100, 0).Add(20*time.Millisecond))
			if (r.RoomObjective.Seconds > 0) != (mode == "inside") {
				t.Fatal("wrong circle charge eligibility")
			}
			if mode == "contested" && !r.RoomObjective.Contested {
				t.Fatal("missing contested indicator")
			}
		})
	}
}
func TestCircleProgressPersistsAndStillRequiresEnemies(t *testing.T) {
	r := circleTestRun()
	if r.RoomObjective == nil {
		t.Fatal("circle missing")
	}
	r.Player.X = r.RoomObjective.Zone.X
	r.Player.Y = r.RoomObjective.Zone.Y
	for n := 0; n < 250; n++ {
		r.tick(Input{}, .02)
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 500; n++ {
		saved.tick(Input{}, .02)
	}
	if !saved.RoomObjective.Complete || saved.RoomObjective.Seconds != 15 || saved.Status != "fighting" {
		t.Fatal("circle duration or enemy gate incorrect")
	}
	count := 0
	for _, e := range saved.Events {
		if e.Kind == "circle_complete" {
			count++
		}
	}
	saved.Enemies[0].HP = 0
	saved.tick(Input{}, .02)
	if count != 1 || saved.Status != "cleared" {
		t.Fatal("completion cue missing or room did not clear")
	}
}

func TestCircleCampaignPlacementAndClearFootprint(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			expected := level.ID%10 == 4 && room == 1
			if (arena.Objective == "hold_circle") != expected {
				t.Fatal("unexpected circle room")
			}
			if !expected {
				continue
			}
			count++
			r := NewRunAtLevel("circle-placement", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level.ID)
			r.Room = room
			r.spawnRoom()
			z := r.RoomObjective.Zone
			for _, o := range arena.solidObstacles() {
				dx := (clamp(z.X, o.X, o.X+o.W) - z.X) / z.RadiusX
				dy := (clamp(z.Y, o.Y, o.Y+o.H) - z.Y) / z.RadiusY
				if dx*dx+dy*dy <= 1 {
					t.Fatal("circle overlaps cover")
				}
			}
			if !r.clearMeleePath(&r.Player, &Actor{X: z.X, Y: z.Y}) {
				t.Fatal("circle approach obstructed")
			}
		}
	}
	if count != 10 {
		t.Fatal("expected one circle room per region")
	}
}
