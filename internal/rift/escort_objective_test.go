package rift

import (
	"encoding/json"
	"strings"
	"testing"
)

func escortTestRun() *Run {
	r := circleTestRun()
	r.Level.Rooms[0].Objective = "escort_spirit"
	r.beginRoomObjective()
	return r
}
func TestSpiritWaitsForEscortAndThreatClearance(t *testing.T) {
	r := escortTestRun()
	o := r.RoomObjective
	if o == nil || o.Escort == nil {
		t.Fatal("spirit missing")
	}
	start := o.Escort.X
	r.tickEscortObjective(.2)
	if o.Escort.X != start || o.EscortMoving {
		t.Fatal("spirit left player behind")
	}
	r.Player.X, r.Player.Y = start, 330
	r.Enemies[0].X, r.Enemies[0].Y = start+50, 330
	r.tickEscortObjective(.2)
	if o.Escort.X != start || !o.Contested {
		t.Fatal("spirit ignored threat")
	}
	r.Enemies[0].X = 1400
	r.Enemies[0].Y = 490
	r.tickEscortObjective(.2)
	if o.Escort.X <= start || !o.EscortMoving || o.Contested {
		t.Fatal("spirit did not resume")
	}
}
func TestSpiritProgressPersistsAndRequiresPatrol(t *testing.T) {
	r := escortTestRun()
	if r.RoomObjective == nil {
		t.Fatal("spirit missing")
	}
	r.Player.X, r.Player.Y = r.RoomObjective.Escort.X, 330
	r.tickEscortObjective(1)
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.RoomObjective.Escort.X != r.RoomObjective.Escort.X {
		t.Fatal("escort position lost")
	}
	for n := 0; n < 1000 && !saved.RoomObjective.Complete; n++ {
		saved.Player.X, saved.Player.Y = saved.RoomObjective.Escort.X, 330
		saved.tick(Input{}, .02)
	}
	if !saved.RoomObjective.Complete || saved.RoomObjective.EscortMoving || saved.Status != "fighting" {
		t.Fatal("arrival or patrol gate incorrect")
	}
	saved.Enemies[0].HP = 0
	saved.tick(Input{}, .02)
	if saved.Status != "cleared" {
		t.Fatal("patrol clear did not complete escort room")
	}
}
func TestSpiritStopsDuringPauseAndDefeat(t *testing.T) {
	for _, mode := range []string{"paused", "dead"} {
		t.Run(mode, func(t *testing.T) {
			r := escortTestRun()
			if r.RoomObjective == nil {
				t.Fatal("spirit missing")
			}
			r.Player.X, r.Player.Y = r.RoomObjective.Escort.X, 330
			start := r.RoomObjective.Escort.X
			if mode == "paused" {
				r.Paused = true
			} else {
				r.Player.HP = 0
			}
			r.tickEscortObjective(1)
			if r.RoomObjective.Escort.X != start {
				t.Fatal("inactive escort moved")
			}
		})
	}
}

func TestSpiritCampaignRoutesStayClear(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			expected := level.ID%10 == 0 && room == 1
			if (arena.Objective == "escort_spirit") != expected {
				t.Fatal("wrong escort placement")
			}
			if !expected {
				continue
			}
			count++
			if !strings.Contains(level.Tactic, "escort route") {
				t.Fatal("missing escort preview")
			}
			for x := 350.0; x <= arena.Exit.X; x += 5 {
				for _, cover := range arena.solidObstacles() {
					if contains(cover, x, 320, 15) {
						t.Fatalf("mission %d spirit path blocked", level.ID)
					}
				}
			}
		}
	}
	if count != 10 {
		t.Fatal("missing escort region")
	}
}
