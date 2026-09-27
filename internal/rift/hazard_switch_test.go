package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func switchTestRun(t *testing.T) *Run {
	t.Helper()
	r := circleTestRun()
	r.RoomObjective = nil
	if err := json.Unmarshal([]byte(`{"name":"Switch room","hazard_switch":{"x":370,"y":475,"charge":0,"used":false},"hazards":[{"x":600,"y":350,"w":90,"h":40,"kind":"fire","period":7,"offset":0,"duration":1,"jumpable":true}]}`), &r.Level.Rooms[0]); err != nil {
		t.Fatal(err)
	}
	r.Player.X, r.Player.Y = 370, 475
	r.Player.Pose = "idle"
	r.Player.PoseTime = 0
	return r
}

func TestHazardSwitchRequiresGuardBetweenAttacks(t *testing.T) {
	for _, mode := range []string{"guard", "idle", "attack", "skill", "jump", "away", "paused"} {
		t.Run(mode, func(t *testing.T) {
			r := switchTestRun(t)
			in := Input{Guard: true}
			switch mode {
			case "idle":
				in.Guard = false
			case "attack":
				in.Attack = true
			case "skill":
				in.Skill = "unknown"
			case "jump":
				in.Jump = true
			case "away":
				r.Player.X = 500
			case "paused":
				r.Paused = true
			}
			for i := 0; i < 40; i++ {
				r.Step(in, time.UnixMilli(r.LastMS+20))
			}
			if r.Arena().Hazards[0].Disabled != (mode == "guard") {
				t.Fatalf("%s switch state=%v", mode, r.Arena().Hazards[0].Disabled)
			}
		})
	}
}

func TestHazardSwitchSavedChargeAndShutdown(t *testing.T) {
	r := switchTestRun(t)
	for i := 0; i < 15; i++ {
		r.Step(Input{Guard: true}, time.UnixMilli(r.LastMS+20))
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		now := time.UnixMilli(r.LastMS + 20)
		r.Step(Input{Guard: true}, now)
		saved.Step(Input{Guard: true}, now)
	}
	if !r.Arena().Hazards[0].Disabled || !saved.Arena().Hazards[0].Disabled {
		t.Fatal("charge or shutdown lost on save")
	}
	count := 0
	for _, e := range saved.Events {
		if e.Kind == "hazard_switch" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("switch cues=%d", count)
	}
}

func TestHazardSwitchInterruptedAndBlocked(t *testing.T) {
	for _, mode := range []string{"release", "wall", "recovery"} {
		t.Run(mode, func(t *testing.T) {
			r := switchTestRun(t)
			for i := 0; i < 15; i++ {
				r.Step(Input{Guard: true}, time.UnixMilli(r.LastMS+20))
			}
			if r.Arena().HazardSwitch.Charge <= 0 {
				t.Fatal("switch did not begin charging")
			}
			in := Input{Guard: true}
			switch mode {
			case "release":
				in.Guard = false
			case "wall":
				r.Player.X = 345
				r.Level.Rooms[0].HighCover = []Obstacle{{355, 450, 5, 40}}
			case "recovery":
				r.Player.Pose = "recovery"
				r.Player.PoseTime = 1
			}
			r.Step(in, time.UnixMilli(r.LastMS+20))
			if r.Arena().HazardSwitch.Charge != 0 || r.Arena().HazardSwitch.Used {
				t.Fatal("interruption did not reset charge")
			}
		})
	}
}

func TestHazardSwitchCampaignPlacementAndDetachedState(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		arena := level.Rooms[0]
		s := arena.HazardSwitch
		if s == nil {
			continue
		}
		count++
		r := circleTestRun()
		r.Level = &level
		r.RoomObjective = nil
		r.Player.X, r.Player.Y = 180, 400
		for i := 0; i < 100; i++ {
			r.moveActor(&r.Player, 0, clamp(s.Y-r.Player.Y, -4, 4), false)
		}
		for i := 0; i < 100; i++ {
			r.moveActor(&r.Player, clamp(s.X-r.Player.X, -4, 4), 0, false)
		}
		if r.Player.X != s.X || r.Player.Y != s.Y {
			t.Fatalf("mission %d switch unreachable from entrance lane", level.ID)
		}
		for i := 0; i < 40; i++ {
			r.Step(Input{Guard: true}, time.UnixMilli(r.LastMS+20))
		}
		if !r.Arena().HazardSwitch.Used {
			t.Fatalf("mission %d switch not usable", level.ID)
		}
		for _, h := range r.Arena().Hazards {
			if !h.Disabled {
				t.Fatal("linked hazard remained live")
			}
		}
		if Campaign()[level.ID-1].Rooms[0].HazardSwitch.Used {
			t.Fatal("mutated authored switch")
		}
	}
	if count != 10 {
		t.Fatalf("switch rooms=%d", count)
	}
}
