package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func TestVanguardPerfectGuardReward(t *testing.T) {
	for _, mode := range []string{"enemy", "rear", "expired", "hazard", "other", "full", "recovery"} {
		t.Run(mode, func(t *testing.T) {
			r := testRun()
			r.Build.Class = "vanguard"
			r.Player.Guard = true
			r.SkillTimers["perfect_guard"] = .22
			r.Resource = 0
			x := r.Player.X + 20*r.Player.Facing
			switch mode {
			case "rear":
				x = r.Player.X - 20*r.Player.Facing
			case "expired":
				r.SkillTimers["perfect_guard"] = 0
			case "other":
				r.Build.Class = "marksman"
			case "full":
				r.Resource = 3
			case "recovery":
				r.SkillTimers["connection_grace"] = 1
			}
			before := r.Resource
			if mode == "hazard" {
				r.hurtPlayerFromHazard(10, x, r.Player.Y)
			} else {
				r.hurtPlayerFromEnemy(10, x, r.Player.Y, "enemy")
			}
			want := before
			if mode == "enemy" {
				want++
			}
			if r.Resource != want {
				t.Fatalf("charges=%d want %d", r.Resource, want)
			}
			if mode != "enemy" {
				return
			}
			raw, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var saved Run
			if err = json.Unmarshal(raw, &saved); err != nil {
				t.Fatal(err)
			}
			saved.hurtPlayerFromEnemy(10, x, saved.Player.Y, "enemy")
			if saved.Resource != 1 {
				t.Fatal("same guard rewarded twice after save")
			}
			saved.Enemies = nil
			saved.Step(Input{}, time.UnixMilli(saved.LastMS+20))
			saved.Step(Input{Guard: true}, time.UnixMilli(saved.LastMS+20))
			saved.hurtPlayerFromEnemy(10, saved.Player.X+20*saved.Player.Facing, saved.Player.Y, "enemy")
			if saved.Resource != 2 {
				t.Fatal("new guard did not earn reward")
			}
		})
	}
}
