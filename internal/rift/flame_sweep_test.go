package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func sweepTestRun() *Run {
	r := circleTestRun()
	r.RoomObjective = nil
	r.Build.Armor = 0
	r.SkillTimers = map[string]float64{}
	r.Level.Rooms[0] = Arena{Hazards: []Hazard{{Obstacle: Obstacle{500, 350, 240, 40}, Kind: "sweeping_flame", Jumpable: true, Period: 7, Duration: 1.8}}}
	r.Player.X, r.Player.Y = 720, 370
	r.Clock = 1.21
	return r
}

func TestFlameSweepOnlyMovingStripDealsDamage(t *testing.T) {
	for _, tc := range []struct {
		phase, x float64
		hit      bool
	}{{1.21, 510, true}, {1.21, 720, false}, {2.1, 620, true}, {2.1, 510, false}, {2.99, 720, true}, {3.01, 720, false}, {.9, 510, false}} {
		r := sweepTestRun()
		r.Clock = tc.phase
		r.Player.X = tc.x
		hp := r.Player.HP
		r.hazardTick()
		if (r.Player.HP < hp) != tc.hit {
			t.Fatalf("phase=%v x=%v hit=%v want=%v", tc.phase, tc.x, r.Player.HP < hp, tc.hit)
		}
	}
}

func TestFlameSweepJumpDodgeAndSavedClock(t *testing.T) {
	for _, mode := range []string{"jump", "dodge"} {
		r := sweepTestRun()
		r.Player.X = 510
		if mode == "jump" {
			r.Player.Jump = .4
		} else {
			r.SkillTimers["dodge_invulnerability"] = .3
		}
		hp := r.Player.HP
		r.hazardTick()
		if r.Player.HP != hp {
			t.Fatalf("%s failed", mode)
		}
	}
	r := sweepTestRun()
	r.Player.X = 620
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
			t.Fatal("saved sweep diverged")
		}
	}
}

func TestCampaignFlameSweeps(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for _, arena := range level.Rooms {
			for _, h := range arena.Hazards {
				if h.Kind == "sweeping_flame" {
					count++
					if level.Region != 1 || !h.Jumpable || h.Duration != 1.8 || h.W != 240 {
						t.Fatal("unexpected sweep")
					}
				}
			}
		}
	}
	if count != 10 {
		t.Fatalf("sweeps=%d", count)
	}
}

func TestFlameSweepActivationCueDoesNotRepeatOnSave(t *testing.T) {
	r := sweepTestRun()
	r.hazardTick()
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		saved.hazardTick()
	}
	count := 0
	for _, e := range saved.Events {
		if e.Kind == "flame_sweep" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("activation cues=%d", count)
	}
}
