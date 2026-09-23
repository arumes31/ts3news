package rift

import (
	"testing"
	"time"
)

func TestPerfectGuardWindowEdges(t *testing.T) {
	for _, mode := range []string{"inside", "expired", "rear", "released", "airborne"} {
		t.Run(mode, func(t *testing.T) {
			r := testRun()
			r.Enemies = []Actor{{ID: "distant", X: 1400, Y: 500, HP: 1000, MaxHP: 1000, Cooldown: 10}}
			now := time.Unix(100, 0)
			r.Step(Input{Guard: true}, now.Add(20*time.Millisecond))
			if r.SkillTimers["perfect_guard"] != .22 {
				t.Fatalf("initial window=%v", r.SkillTimers["perfect_guard"])
			}
			r.Step(Input{Guard: true}, now.Add(150*time.Millisecond))
			if r.SkillTimers["perfect_guard"] >= .22 {
				t.Fatal("holding guard renewed window")
			}
			sourceX := r.Player.X + 20*r.Player.Facing
			want := "perfect_guard"
			switch mode {
			case "expired":
				r.Step(Input{Guard: true}, now.Add(300*time.Millisecond))
				want = "block"
			case "rear":
				sourceX = r.Player.X - 20*r.Player.Facing
				want = "hurt"
			case "released":
				r.Step(Input{}, now.Add(170*time.Millisecond))
				want = "hurt"
			case "airborne":
				r.Player.Jump = .5
				r.Step(Input{Guard: true}, now.Add(170*time.Millisecond))
				want = "hurt"
			}
			if (mode == "released" || mode == "airborne") && r.SkillTimers["perfect_guard"] != 0 {
				t.Fatal("inactive guard retained perfect window")
			}
			r.hurtPlayer(30, sourceX, r.Player.Y)
			if got := r.Events[len(r.Events)-1].Kind; got != want {
				t.Fatalf("hit cue=%s, want %s", got, want)
			}
		})
	}
}
