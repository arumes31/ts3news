package rift

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func beaconTestRun() *Run {
	r := circleTestRun()
	r.Level.Rooms[0].Objective = "moving_beacons"
	r.beginRoomObjective()
	return r
}
func TestBeaconMovesAndSavesPartialCapture(t *testing.T) {
	r := beaconTestRun()
	o := r.RoomObjective
	if o == nil {
		t.Fatal("beacon missing")
	}
	start := o.Zone.X
	r.Player.X, r.Player.Y = o.Zone.X, o.Zone.Y
	r.tickBeaconObjective(1)
	if o.Zone.X == start || o.Seconds != 1 || o.BeaconTime != 1 {
		t.Fatal("beacon did not move and charge")
	}
	r.Player.X = 100
	r.tickBeaconObjective(.5)
	if o.Seconds != 1 || o.Charging {
		t.Fatal("leaving reset or advanced charge")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.RoomObjective.Seconds != 1 || saved.RoomObjective.BeaconTime != 1.5 || saved.RoomObjective.Zone.X != o.Zone.X {
		t.Fatal("capture state lost")
	}
	for n := 0; n < 600 && !saved.RoomObjective.Complete; n++ {
		saved.Player.X, saved.Player.Y = saved.RoomObjective.Zone.X, saved.RoomObjective.Zone.Y
		saved.tick(Input{}, .02)
	}
	if !saved.RoomObjective.Complete || saved.RoomObjective.Collected != 3 || saved.Status != "fighting" {
		t.Fatal("three captures or patrol gate incorrect")
	}
	captures, completions := 0, 0
	for _, event := range saved.Events {
		if event.Kind == "beacon_captured" {
			captures++
		}
		if event.Kind == "beacons_complete" {
			completions++
		}
	}
	if captures != 3 || completions != 1 {
		t.Fatal("capture/completion events duplicated or missing")
	}
	saved.Enemies[0].HP = 0
	saved.tick(Input{}, .02)
	if saved.Status != "cleared" {
		t.Fatal("beacons failed to clear")
	}
}
func TestBeaconPauseDeathAndAirborneRules(t *testing.T) {
	for _, mode := range []string{"paused", "dead", "airborne"} {
		t.Run(mode, func(t *testing.T) {
			r := beaconTestRun()
			if r.RoomObjective == nil {
				t.Fatal("beacon missing")
			}
			r.Player.X, r.Player.Y = r.RoomObjective.Zone.X, r.RoomObjective.Zone.Y
			switch mode {
			case "paused":
				r.Paused = true
			case "dead":
				r.Player.HP = 0
			case "airborne":
				r.Player.Jump = .5
			}
			r.tickBeaconObjective(.2)
			if r.RoomObjective.Seconds != 0 {
				t.Fatal("inactive or airborne capture")
			}
			if mode != "airborne" && r.RoomObjective.BeaconTime != 0 {
				t.Fatal("inactive beacon moved")
			}
		})
	}
}

func TestBeaconCampaignPathsStayClear(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			expected := level.ID%10 == 2 && room == 1
			if (arena.Objective == "moving_beacons") != expected {
				t.Fatal("wrong beacon placement")
			}
			if !expected {
				continue
			}
			count++
			if !strings.Contains(level.Tactic, "capture three moving beacons") {
				t.Fatal("missing preview")
			}
			for _, center := range beaconCenters {
				for sample := 0; sample < 128; sample++ {
					x, y := center[0]+100*math.Sin(float64(sample)*math.Pi/64), center[1]
					for _, cover := range arena.solidObstacles() {
						if contains(cover, x, y, 10) {
							t.Fatalf("mission %d beacon path inside cover", level.ID)
						}
					}
				}
			}
			// These bypasses connect the three paths without crossing the central well.
			for _, point := range [][2]float64{{600, 330}, {600, 480}, {1100, 480}, {1100, 330}} {
				for _, cover := range arena.solidObstacles() {
					if contains(cover, point[0], point[1], 10) {
						t.Fatal("beacon approach blocked")
					}
				}
			}
		}
	}
	if count != 10 {
		t.Fatal("missing beacon region")
	}
}
