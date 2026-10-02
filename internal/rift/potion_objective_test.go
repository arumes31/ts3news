package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestNoPotionObjectiveUsesConfirmedMissionCounter(t *testing.T) {
	for _, mode := range []string{"unused", "rejected", "healing_skill", "used"} {
		t.Run(mode, func(t *testing.T) {
			r := NewRunAtLevel("potion-goal", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
			goal := func(run *Run) *ObjectiveProgress {
				for i := range run.Objectives.Entries {
					if run.Objectives.Entries[i].ID == "no_potions" {
						return &run.Objectives.Entries[i]
					}
				}
				t.Fatal("no-potion objective missing")
				return nil
			}
			switch mode {
			case "rejected":
				if err := r.UseHealingPotion(50); err == nil {
					t.Fatal("full-health potion accepted")
				}
			case "healing_skill":
				r.Player.HP -= 30
				r.healPlayerBySkill(20, "heal")
			case "used":
				r.Player.HP -= 30
				if err := r.UseHealingPotion(50); err != nil {
					t.Fatal(err)
				}
			}
			raw, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var saved Run
			if err = json.Unmarshal(raw, &saved); err != nil {
				t.Fatal(err)
			}
			r = &saved
			r.UpdateObjectives()
			want := "active"
			if mode == "used" {
				want = "failed"
			}
			if goal(r).Status != want {
				t.Fatalf("saved objective: %+v", goal(r))
			}
			r.Room = 2
			r.Status = "cleared"
			r.FinishCheckpoint("bank", nil)
			completed := 0
			if mode != "used" {
				completed = 1
			}
			if r.ObjectiveHistory["Wayfarer"]["no_potions"] != completed {
				t.Fatal("wrong completion record")
			}
			r.setLevel(2, content.AbyssMobCatalog())
			r.UpdateObjectives()
			if goal(r).Status != "active" || goal(r).Current != 0 {
				t.Fatal("previous mission potion polluted fresh objective")
			}
			r.Room = 2
			r.Status = "cleared"
			r.FinishCheckpoint("bank", nil)
			if r.ObjectiveHistory["Wayfarer"]["no_potions"] != completed+1 {
				t.Fatal("fresh mission completion missing")
			}
		})
	}
}
