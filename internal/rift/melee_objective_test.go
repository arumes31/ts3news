package rift

import (
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestMeleeOnlyObjective(t *testing.T) {
	for _, kind := range []string{"slash", "heal", "shield", "fire", "ice", "quake", "ultimate"} {
		t.Run(kind, func(t *testing.T) {
			build := testRun().Build
			build.Skills = []Skill{{ID: "check", Kind: kind, Cost: 10, Power: 1}}
			r := NewRunAtLevel("melee-goal", build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
			r.Enemies = nil
			var goal *ObjectiveProgress
			for i := range r.Objectives.Entries {
				if r.Objectives.Entries[i].ID == "melee_only" {
					goal = &r.Objectives.Entries[i]
				}
			}
			if goal == nil {
				t.Fatal("melee objective missing")
			}
			r.Player.Mana = 0
			r.cast("check")
			r.UpdateObjectives()
			if goal.Current != 0 || goal.Status != "active" {
				t.Fatal("rejected cast failed objective")
			}
			r.Player.Mana = 100
			r.cast("check")
			r.UpdateObjectives()
			allowed := kind == "slash" || kind == "heal" || kind == "shield"
			if (goal.Status == "active") != allowed {
				t.Fatalf("incorrect classification for %s: %+v", kind, goal)
			}
			r.Room = 2
			r.Status = "cleared"
			r.FinishCheckpoint("bank", nil)
			want := 0
			if allowed {
				want = 1
			}
			if r.ObjectiveHistory["Wayfarer"]["melee_only"] != want {
				t.Fatal("wrong melee-only completion")
			}
			r.setLevel(2, content.AbyssMobCatalog())
			r.Room = 2
			r.Status = "cleared"
			r.FinishCheckpoint("bank", nil)
			if r.ObjectiveHistory["Wayfarer"]["melee_only"] != want+1 {
				t.Fatal("previous mission casts polluted new mission")
			}
		})
	}
}
