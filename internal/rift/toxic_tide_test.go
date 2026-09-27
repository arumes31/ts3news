package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestToxicTideOptionalRoomAndMovingPuddles(t *testing.T) {
	r, err := NewPracticeRun("toxic", testRun().Build, "toxic_tide", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Arena().Hazards) != 3 || r.Practice.GoalX != 1350 {
		t.Fatal("challenge layout missing")
	}
	h := r.Arena().Hazards[0]
	for _, tc := range []struct{ phase, x float64 }{{1.2, h.X}, {3.1, h.X + h.W - 80}, {5, h.X}} {
		b := h.ContactBounds(tc.phase)
		if math.Abs(b.X-tc.x) > 1e-7 || b.W != 80 {
			t.Fatalf("moving pool phase=%v bounds=%+v", tc.phase, b)
		}
	}
	r.Clock = 1.21
	r.Player.X = h.X + h.W - 10
	r.Player.Y = h.Y + h.H/2
	r.SkillTimers = map[string]float64{}
	hp := r.Player.HP
	r.hazardTick()
	if r.Player.HP != hp {
		t.Fatal("empty side of pool envelope dealt damage")
	}
	r.Player.X = h.X + 20
	r.hazardTick()
	if r.Player.HP >= hp || r.SlowSource != "poison" {
		t.Fatal("moving pool did not damage and slow")
	}
}

func TestToxicTideSaveResetAndCompletion(t *testing.T) {
	r, err := NewPracticeRun("toxic", testRun().Build, "toxic_tide", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	r.Clock = 2.3
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
		r.Step(Input{X: 1, Jump: true}, now)
		saved.Step(Input{X: 1, Jump: true}, now)
		if r.Player.X != saved.Player.X || r.Player.HP != saved.Player.HP {
			t.Fatal("saved challenge diverged")
		}
	}
	r.Player.X = r.Practice.GoalX
	r.practiceTick()
	if r.Status != "complete" || !r.Practice.Completed || r.Gold != 0 {
		t.Fatal("challenge completion failed")
	}
	if err = r.ResetPractice(time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	if r.Practice.Completed || r.Clock != 0 || len(r.Arena().Hazards) != 3 {
		t.Fatal("challenge did not reset")
	}
}
