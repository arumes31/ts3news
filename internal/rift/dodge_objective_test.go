package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestLimitedDodgeObjective(t *testing.T) {
	for _, count := range []int{0, 3, 4} {
		r := NewRunAtLevel("dodge-goal", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
		r.Level.Rooms[0].Hazards = nil
		r.Level.Rooms[0].HighCover = nil
		r.Enemies = []Actor{{ID: "far", Kind: "goblin", X: 1400, Y: 500, HP: 100, MaxHP: 100, Knockdown: 100}}
		r.tick(Input{Jump: true}, .02)
		for i := 0; i < count; i++ {
			r.Player.Jump = .5
			r.Projectiles = []Projectile{{Enemy: true, X: r.Player.X, Y: r.Player.Y, Life: 1, Power: 10}}
			r.tick(Input{}, .02)
		}
		var goal *ObjectiveProgress
		for i := range r.Objectives.Entries {
			if r.Objectives.Entries[i].ID == "limited_dodge" {
				goal = &r.Objectives.Entries[i]
			}
		}
		if goal == nil {
			t.Fatal("dodge objective missing")
		}
		if int(goal.Current) != count || (goal.Status == "failed") != (count > 3) {
			t.Fatalf("wrong progress for %d: %+v", count, goal)
		}
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved Run
		if err = json.Unmarshal(data, &saved); err != nil {
			t.Fatal(err)
		}
		saved.Room = 2
		saved.Status = "cleared"
		saved.FinishCheckpoint("bank", nil)
		want := 1
		if count > 3 {
			want = 0
		}
		if saved.ObjectiveHistory["Wayfarer"]["limited_dodge"] != want {
			t.Fatal("incorrect dodge completion")
		}
		saved.setLevel(2, content.AbyssMobCatalog())
		saved.Room = 2
		saved.Status = "cleared"
		saved.FinishCheckpoint("bank", nil)
		if saved.ObjectiveHistory["Wayfarer"]["limited_dodge"] != want+1 {
			t.Fatal("previous mission dodges carried over")
		}
	}
}

func TestDodgeCounterIncludesMeleeAndBossSlam(t *testing.T) {
	for _, kind := range []string{"goblin", "boss"} {
		r := testRun()
		r.Player.Jump = .5
		r.Enemies = []Actor{{ID: "attacker", Kind: kind, HP: 100, MaxHP: 100, X: r.Player.X + 40, Y: r.Player.Y, Windup: .01, TargetX: r.Player.X, TargetY: r.Player.Y}}
		r.enemyTick(0, .02)
		if r.Stats.Dodges != 1 || r.Stats.DamageTaken != 0 {
			t.Fatalf("%s dodge not recorded", kind)
		}
	}
}
