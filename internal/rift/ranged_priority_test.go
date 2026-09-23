package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestRangedPriorityObjective(t *testing.T) {
	for _, mode := range []string{"absent", "correct", "wrong", "wound"} {
		t.Run(mode, func(t *testing.T) {
			r := NewRunAtLevel("priority", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
			r.Enemies = []Actor{{ID: "ranged", Kind: "archer", HP: 50, MaxHP: 50}, {ID: "melee", Kind: "goblin", HP: 50, MaxHP: 50}}
			if mode == "absent" {
				r.Enemies[0].Kind = "goblin"
			}
			r.EncounterPlan = [][]Actor{append([]Actor{}, r.Enemies...)}
			r.beginObjectives()
			var goal *ObjectiveProgress
			for i := range r.Objectives.Entries {
				if r.Objectives.Entries[i].ID == "ranged_priority" {
					goal = &r.Objectives.Entries[i]
				}
			}
			if mode == "absent" {
				if goal != nil {
					t.Fatal("goal offered without ranged enemy")
				}
				return
			}
			if goal == nil {
				t.Fatal("priority objective missing")
			}
			switch mode {
			case "correct":
				r.hurtEnemy(0, 100, "hit")
				r.hurtEnemy(1, 100, "hit")
			case "wrong":
				r.hurtEnemy(1, 100, "hit")
				r.hurtEnemy(0, 100, "hit")
			case "wound":
				r.hurtEnemy(1, 1, "hit")
				r.hurtEnemy(0, 100, "hit")
				r.hurtEnemy(1, 100, "hit")
			}
			r.UpdateObjectives()
			failed := mode == "wrong"
			if (goal.Status == "failed") != failed {
				t.Fatalf("wrong priority result %+v", goal)
			}
			if r.Status != "fighting" {
				t.Fatal("optional failure ended combat")
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
			if failed {
				want = 0
			}
			if saved.ObjectiveHistory["Wayfarer"]["ranged_priority"] != want {
				t.Fatal("wrong banked priority result")
			}
		})
	}
}

func TestRangedPriorityEligibilityAcrossCampaign(t *testing.T) {
	for level := 1; level <= LevelCount; level++ {
		r := NewRunAtLevel("ranged-eligibility", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level)
		planned, offered := false, false
		for _, room := range r.EncounterPlan {
			for _, enemy := range room {
				planned = planned || enemy.Kind == "archer"
			}
		}
		for _, entry := range r.Objectives.Entries {
			offered = offered || entry.ID == "ranged_priority"
		}
		if planned != offered {
			t.Fatalf("wrong eligibility at mission %d", level)
		}
	}
}
