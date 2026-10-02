package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func frostTestRun() *Run {
	r := circleTestRun()
	r.RoomObjective = nil
	r.Enemies = []Actor{{ID: "far", Kind: "goblin", X: 1500, Y: 400, HP: 100, Knockdown: 100}}
	r.Level.Rooms[0] = Arena{Hazards: []Hazard{{Obstacle: Obstacle{400, 330, 400, 160}, Kind: "ice", Period: 7, Duration: 3}}}
	// Decode the opt-in field so the first test can fail before the field exists.
	_ = json.Unmarshal([]byte(`{"x":400,"y":330,"w":400,"h":160,"kind":"ice","period":7,"duration":3,"jumpable":true,"slippery":true}`), &r.Level.Rooms[0].Hazards[0])
	r.Clock = 1.3
	r.Player.X, r.Player.Y = 500, 400
	r.Player.Vx, r.Player.Vy = 235, 0
	r.SkillTimers["hazard-hit"] = 10
	return r
}

func TestFrostReleaseBrakesGraduallyAndStops(t *testing.T) {
	r := frostTestRun()
	r.Step(Input{}, time.UnixMilli(r.LastMS+20))
	if r.Player.X <= 500 || r.Player.Vx <= 0 || r.Player.Vx >= 235 {
		t.Fatalf("no controlled slide: x=%v vx=%v", r.Player.X, r.Player.Vx)
	}
	for i := 0; i < 25; i++ {
		r.Step(Input{}, time.UnixMilli(r.LastMS+20))
	}
	if r.Player.Vx != 0 || r.Player.X > 550 {
		t.Fatalf("unbounded slide: x=%v vx=%v", r.Player.X, r.Player.Vx)
	}
}

func TestFrostCounterplayAndInactiveGround(t *testing.T) {
	for _, name := range []string{"guard", "jump", "airborne", "disabled", "inactive", "outside", "legacy", "cleared"} {
		t.Run(name, func(t *testing.T) {
			r := frostTestRun()
			in := Input{}
			switch name {
			case "guard":
				in.Guard = true
			case "jump":
				in.Jump = true
			case "airborne":
				r.Player.Jump = .4
			case "disabled":
				r.Level.Rooms[0].Hazards[0].Disabled = true
			case "inactive":
				r.Clock = 5
			case "outside":
				r.Player.X = 300
			case "legacy":
				_ = json.Unmarshal([]byte(`{"x":400,"y":330,"w":400,"h":160,"kind":"ice","period":7,"duration":3}`), &r.Level.Rooms[0].Hazards[0])
			case "cleared":
				r.Status = "cleared"
			}
			x := r.Player.X
			r.Step(in, time.UnixMilli(r.LastMS+20))
			if r.Player.X != x || r.Player.Vx != 0 {
				t.Fatalf("%s should stop slide: x=%v vx=%v", name, r.Player.X, r.Player.Vx)
			}
		})
	}
}

func TestFrostSavedSlideAndReversal(t *testing.T) {
	r := frostTestRun()
	r.Step(Input{X: -1}, time.UnixMilli(r.LastMS+20))
	if r.Player.Vx <= 0 {
		t.Fatal("reversal must first brake momentum")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30; i++ {
		now := time.UnixMilli(r.LastMS + 20)
		r.Step(Input{X: -1}, now)
		saved.Step(Input{X: -1}, now)
		if math.Abs(r.Player.X-saved.Player.X) > 1e-9 || r.Player.Vx != saved.Player.Vx {
			t.Fatal("saved slide diverged")
		}
	}
	if r.Player.Vx >= 0 {
		t.Fatal("opposite input never overcame momentum")
	}
}

func TestFrostCollisionConsumesMomentum(t *testing.T) {
	r := frostTestRun()
	r.Level.Rooms[0].Obstacles = []Obstacle{{513, 350, 50, 100}}
	r.Step(Input{}, time.UnixMilli(r.LastMS+20))
	if r.Player.X != 500 || r.Player.Vx != 0 {
		t.Fatalf("wall retained momentum: x=%v vx=%v", r.Player.X, r.Player.Vx)
	}
}

func TestFrostDiagonalAccelerationAndDodge(t *testing.T) {
	r := frostTestRun()
	r.Player.Vx = 0
	r.Step(Input{X: 1, Y: 1}, time.UnixMilli(r.LastMS+20))
	if r.Player.Vx <= 0 || r.Player.Vx > 12.001 || r.Player.Vy <= 0 || r.Player.Vy > 7.201 {
		t.Fatalf("acceleration: %v,%v", r.Player.Vx, r.Player.Vy)
	}
	r.Step(Input{X: -1, Dodge: true}, time.UnixMilli(r.LastMS+20))
	if r.Player.Vx != -340 {
		t.Fatalf("dodge lost direct control: %v", r.Player.Vx)
	}
}

func TestCampaignIceOptsIntoTractionAndLegacyDoesNot(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for _, arena := range level.Rooms {
			for _, h := range arena.Hazards {
				if h.Kind == "ice" {
					count++
					if !h.Slippery {
						t.Fatalf("mission %d missing frost", level.ID)
					}
				} else if h.Slippery {
					t.Fatal("non-ice has frost traction")
				}
			}
		}
	}
	if count == 0 {
		t.Fatal("no campaign frost patches")
	}
	t.Logf("%d campaign frost patches", count)
}

func TestFrostUnavailableJumpDoesNotBypassTraction(t *testing.T) {
	r := frostTestRun()
	r.SkillTimers["jump"] = 1
	r.Step(Input{Jump: true}, time.UnixMilli(r.LastMS+20))
	if r.Player.Jump != 0 || r.Player.Vx <= 0 || r.Player.X <= 500 {
		t.Fatal("unavailable jump bypassed traction")
	}
}
