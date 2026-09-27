package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func waterTestRun(t *testing.T) *Run {
	t.Helper()
	r := circleTestRun()
	r.RoomObjective = nil
	if err := json.Unmarshal([]byte(`{"name":"Tidal channel","water_currents":[{"x":400,"y":350,"w":500,"h":100,"vx":35,"vy":0}]}`), &r.Level.Rooms[0]); err != nil {
		t.Fatal(err)
	}
	r.Player.X, r.Player.Y = 500, 400
	r.Enemies = []Actor{{ID: "target", Kind: "goblin", X: 700, Y: 400, HP: 100, MaxHP: 100, Cooldown: 100, Pose: "stagger", PoseTime: 100}}
	return r
}

func TestWaterCurrentPushesGroundedFighters(t *testing.T) {
	r := waterTestRun(t)
	r.Step(Input{}, time.UnixMilli(r.LastMS+20))
	if math.Abs(r.Player.X-500.7) > 1e-8 || math.Abs(r.Enemies[0].X-700.7) > 1e-8 {
		t.Fatalf("current movement: player=%v enemy=%v", r.Player.X, r.Enemies[0].X)
	}
}

func TestWaterCurrentCounterplay(t *testing.T) {
	for _, mode := range []string{"guard", "jump", "outside", "paused", "cleared", "dead"} {
		t.Run(mode, func(t *testing.T) {
			r := waterTestRun(t)
			in := Input{}
			switch mode {
			case "guard":
				in.Guard = true
			case "jump":
				in.Jump = true
			case "outside":
				r.Player.Y = 320
			case "paused":
				r.Paused = true
			case "cleared":
				r.Status = "cleared"
			case "dead":
				r.Player.HP = 0
			}
			r.Step(in, time.UnixMilli(r.LastMS+20))
			if r.Player.X != 500 {
				t.Fatalf("%s did not avoid drift: %v", mode, r.Player.X)
			}
		})
	}
	r := waterTestRun(t)
	r.Step(Input{X: -1}, time.UnixMilli(r.LastMS+20))
	if r.Player.X >= 500 {
		t.Fatal("walking upstream failed")
	}
}

func TestWaterCurrentSavedContinuation(t *testing.T) {
	r := waterTestRun(t)
	r.Step(Input{}, time.UnixMilli(r.LastMS+20))
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
		r.Step(Input{}, now)
		saved.Step(Input{}, now)
		if r.Player.X != saved.Player.X || r.Enemies[0].X != saved.Enemies[0].X {
			t.Fatal("saved current diverged")
		}
	}
}

func TestWaterCurrentCollisionAndGroundExclusions(t *testing.T) {
	r := waterTestRun(t)
	r.Level.Rooms[0].HighCover = []Obstacle{{510.5, 350, 20, 100}}
	r.Step(Input{}, time.UnixMilli(r.LastMS+20))
	if r.Player.X != 500 {
		t.Fatal("current crossed wall")
	}
	for _, mode := range []string{"flying", "burrowed", "elevated", "knockdown", "dead"} {
		t.Run(mode, func(t *testing.T) {
			r := waterTestRun(t)
			a := &r.Enemies[0]
			switch mode {
			case "flying":
				a.Flying = true
			case "burrowed":
				a.Burrowed = true
			case "elevated":
				a.Elevation = 16
			case "knockdown":
				a.Knockdown = 1
			case "dead":
				a.HP = 0
			}
			r.waterCurrentTick(.02)
			if a.X != 700 {
				t.Fatal("ungrounded enemy drifted")
			}
		})
	}
}

func TestWaterCurrentCuesAndCampaignCopies(t *testing.T) {
	r := waterTestRun(t)
	seen := map[int]bool{}
	cues := 0
	for i := 0; i < 20; i++ {
		r.Step(Input{}, time.UnixMilli(r.LastMS+20))
		for _, e := range r.Events {
			if e.Kind == "water_current" && !seen[e.ID] {
				cues++
				seen[e.ID] = true
			}
		}
	}
	if cues != 1 {
		t.Fatalf("entry cues=%d", cues)
	}
	r.Step(Input{Guard: true}, time.UnixMilli(r.LastMS+20))
	r.Step(Input{}, time.UnixMilli(r.LastMS+20))
	for _, e := range r.Events {
		if e.Kind == "water_current" && !seen[e.ID] {
			cues++
			seen[e.ID] = true
		}
	}
	if cues != 2 {
		t.Fatal("reentering flow did not cue")
	}
	levels := Campaign()
	count := 0
	for _, level := range levels {
		for _, arena := range level.Rooms {
			for _, c := range arena.WaterCurrents {
				count++
				if level.Region != 5 || math.Abs(c.VX) != 35 {
					t.Fatal("unexpected current authoring")
				}
			}
		}
	}
	if count != 10 {
		t.Fatalf("currents=%d", count)
	}
	levels[50].Rooms[0].WaterCurrents[0].VX = 1
	if Campaign()[50].Rooms[0].WaterCurrents[0].VX == 1 {
		t.Fatal("shared current slice")
	}
	r.Level.Rooms[0].WaterCurrents = append(r.Level.Rooms[0].WaterCurrents, r.Level.Rooms[0].WaterCurrents[0])
	vx, vy := r.currentVelocity(&r.Player)
	if math.Hypot(vx, vy) > 40 {
		t.Fatal("overlap exceeded cap")
	}
}
