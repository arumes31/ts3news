package rift

import (
	"encoding/json"
	"testing"
)

func TestLightningTracksLocksAndSurvivesSave(t *testing.T) {
	r := rockTestRun()
	r.Level.Rooms[0].Hazards[0].Kind = "tracking_lightning"
	r.Clock = .1
	r.hazardTick()
	h := r.Arena().Hazards[0]
	if h.X+h.W/2 != r.Player.X || h.Y+h.H/2 != r.Player.Y {
		t.Fatal("marker did not track")
	}
	r.Player.X += 100
	r.Clock = .4
	r.hazardTick()
	h = r.Arena().Hazards[0]
	if h.X+h.W/2 != r.Player.X {
		t.Fatal("marker stopped too early")
	}
	r.Player.X += 150
	r.Clock = .7
	r.hazardTick()
	if r.Arena().Hazards[0].X != h.X {
		t.Fatal("locked marker moved")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	hp := saved.Player.HP
	saved.Clock = 1.21
	saved.hazardTick()
	if saved.Player.HP != hp {
		t.Fatal("strike followed player after lock")
	}
	saved.Player.X = h.X + h.W/2
	saved.Player.Y = h.Y + h.H/2
	saved.hazardTick()
	if saved.Player.HP >= hp {
		t.Fatal("saved strike failed to damage")
	}
	saved.Clock = 7.1
	saved.Player.X = 800
	saved.hazardTick()
	if saved.Arena().Hazards[0].X+h.W/2 != 800 {
		t.Fatal("next cycle failed to track")
	}
}
func TestLightningPausedAndDisabledMarkersStayStill(t *testing.T) {
	for _, mode := range []string{"paused", "disabled"} {
		r := rockTestRun()
		r.Level.Rooms[0].Hazards[0].Kind = "tracking_lightning"
		r.Clock = .2
		before := r.Arena().Hazards[0]
		if mode == "paused" {
			r.Paused = true
		} else {
			r.Level.Rooms[0].Hazards[0].Disabled = true
		}
		r.hazardTick()
		if r.Arena().Hazards[0].X != before.X {
			t.Fatal(mode + " marker moved")
		}
	}
}

func TestLightningCampaignAndCounterplay(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			for _, h := range arena.Hazards {
				if h.Kind != "tracking_lightning" {
					continue
				}
				count++
				if level.Region != 3 || room != 0 || h.Jumpable || h.Duration != .18 || h.Offset != 0 {
					t.Fatal("wrong authored lightning")
				}
			}
		}
	}
	if count != 10 {
		t.Fatal("missing Storm lightning")
	}
	for _, mode := range []string{"jump", "dodge", "move"} {
		r := rockTestRun()
		r.Level.Rooms[0].Hazards[0].Kind = "tracking_lightning"
		r.Clock = 1.21
		hp := r.Player.HP
		if mode == "jump" {
			r.Player.Jump = .5
		}
		if mode == "dodge" {
			r.SkillTimers["dodge_invulnerability"] = .2
		}
		if mode == "move" {
			r.Player.X = 700
		}
		r.hazardTick()
		r.hazardTick()
		if (r.Player.HP < hp) != (mode == "jump") {
			t.Fatal("incorrect lightning counterplay: " + mode)
		}
		cues := 0
		for _, e := range r.Events {
			if e.Kind == "lightning_strike" {
				cues++
			}
		}
		if cues != 1 {
			t.Fatal("strike cue must play once even on miss")
		}
	}
}

func TestLightningMarkerClampedToArena(t *testing.T) {
	for _, point := range [][2]float64{{35, 315}, {1565, 490}} {
		r := rockTestRun()
		r.Level.Rooms[0].Hazards[0].Kind = "tracking_lightning"
		r.Clock = .2
		r.Player.X, r.Player.Y = point[0], point[1]
		r.hazardTick()
		h := r.Arena().Hazards[0]
		if h.X < 35 || h.Y < 315 || h.X+h.W > 1565 || h.Y+h.H > 490 {
			t.Fatal("tracking box escaped arena")
		}
	}
	r := rockTestRun()
	before := r.Arena().Hazards[0]
	r.Clock = .2
	r.hazardTick()
	if r.Arena().Hazards[0] != before {
		t.Fatal("legacy hazard moved")
	}
}
