package rift

import (
	"encoding/json"
	"testing"
	"ts3news/internal/content"
)

func TestHazardAvoidanceObjective(t *testing.T) {
	for _, mode := range []string{"warning", "jump", "outside", "hit", "shield", "dead", "enemy"} {
		t.Run(mode, func(t *testing.T) {
			r := overlappingHazardRun()
			switch mode {
			case "warning":
				r.Clock = .5
			case "jump":
				r.Player.Jump = .5
			case "outside":
				r.Player.X = 500
			case "shield":
				r.Barrier = 100
			case "enemy":
				r.Player.X = 500
				r.hurtPlayer(20, 530, r.Player.Y)
			case "dead":
				r.Player.HP = 0
			}
			r.hazardTick()
			r.UpdateObjectives()
			var goal *ObjectiveProgress
			for i := range r.Objectives.Entries {
				if r.Objectives.Entries[i].ID == "hazard_avoidance" {
					goal = &r.Objectives.Entries[i]
				}
			}
			if goal == nil {
				t.Fatal("hazard objective missing")
			}
			want := mode == "hit" || mode == "shield"
			if (goal.Status == "failed") != want {
				t.Fatalf("wrong hazard outcome: %+v", goal)
			}
			if want && goal.Current != 1 {
				t.Fatal("overlap counted more than once")
			}
			if mode == "shield" && r.Stats.DamageTaken != 0 {
				t.Fatal("shield control did not absorb damage")
			}
			data, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var saved Run
			if err = json.Unmarshal(data, &saved); err != nil {
				t.Fatal(err)
			}
			saved.setLevel(2, content.AbyssMobCatalog())
			saved.Room = 2
			saved.Status = "cleared"
			saved.FinishCheckpoint("bank", nil)
			if saved.ObjectiveHistory["Wayfarer"]["hazard_avoidance"] != 1 {
				t.Fatal("safe mission not banked independently of previous hazards")
			}
		})
	}
}
