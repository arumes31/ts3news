package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func bladeTestRun() *Run {
	r := circleTestRun()
	r.RoomObjective = nil
	r.SkillTimers = map[string]float64{}
	r.Build.Armor = 0
	r.Level.Rooms[0] = Arena{Hazards: []Hazard{{Obstacle: Obstacle{500, 350, 140, 70}, Kind: "rotating_blade", Jumpable: true, Period: 7, Duration: 2.4}}}
	r.Player.Y = 385
	r.Clock = 1.2
	return r
}

func TestRotatingBladeHasSafeHubAndPredictableCircuit(t *testing.T) {
	for _, tc := range []struct{ phase, x, y float64 }{{1.2, 630, 385}, {1.8, 570, 410}, {2.4, 510, 385}, {3, 570, 360}} {
		r := bladeTestRun()
		r.Clock = tc.phase
		b := r.Arena().Hazards[0].ContactBounds(r.Clock)
		if math.Abs(b.X+b.W/2-tc.x) > 1e-7 || math.Abs(b.Y+b.H/2-tc.y) > 1e-7 || b.W != 20 || b.H != 20 {
			t.Fatalf("phase%v bounds%+v", tc.phase, b)
		}
		r.Player.X, r.Player.Y = 570, 385
		hp := r.Player.HP
		r.hazardTick()
		if r.Player.HP != hp {
			t.Fatal("hub must be safe")
		}
		r.Player.X, r.Player.Y = tc.x, tc.y
		r.hazardTick()
		if r.Player.HP >= hp {
			t.Fatal("blade missed its contact square")
		}
	}
}

func TestRotatingBladeEvasionAndSavedClock(t *testing.T) {
	for _, mode := range []string{"jump", "dodge", "warning", "rest"} {
		r := bladeTestRun()
		r.Player.X = 630
		switch mode {
		case "jump":
			r.Player.Jump = .4
		case "dodge":
			r.SkillTimers["dodge_invulnerability"] = .3
		case "warning":
			r.Clock = .8
		case "rest":
			r.Clock = 4
		}
		hp := r.Player.HP
		r.hazardTick()
		if r.Player.HP != hp {
			t.Fatalf("%s took damage", mode)
		}
	}
	r := bladeTestRun()
	r.Player.X = 510
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		now := time.UnixMilli(r.LastMS + 20)
		r.Step(Input{}, now)
		saved.Step(Input{}, now)
		if r.Player.HP != saved.Player.HP || r.Clock != saved.Clock {
			t.Fatal("saved blade diverged")
		}
	}
}

func TestCampaignRotatingBlades(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for _, arena := range level.Rooms {
			for _, h := range arena.Hazards {
				if h.Kind == "rotating_blade" {
					count++
					if level.Region != 6 || !h.Jumpable || h.Duration != 2.4 || h.W != 140 || h.H != 70 {
						t.Fatal("unexpected blade authoring")
					}
				}
			}
		}
	}
	if count != 10 {
		t.Fatalf("blades=%d", count)
	}
}
