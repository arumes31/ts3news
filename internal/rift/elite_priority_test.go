package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestElitePriorityUsesAbyssTier(t *testing.T) {
	for _, tier := range []content.MobType{content.MobEliteMinion, content.MobElite, content.MobMiniboss, content.MobBoss, content.MobCommon} {
		for _, wrong := range []bool{false, true} {
			r := NewRunAtLevel("elite-priority", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
			elite := AdaptMonster(content.Mob{Name: "Test mage", Type: tier})
			elite.HP = 50
			elite.MaxHP = 50
			r.Enemies = []Actor{elite, {ID: "normal", Kind: "goblin", HP: 50, MaxHP: 50}}
			r.EncounterPlan = [][]Actor{append([]Actor{}, r.Enemies...)}
			r.beginObjectives()
			var goal *ObjectiveProgress
			for i := range r.Objectives.Entries {
				if r.Objectives.Entries[i].ID == "elite_priority" {
					goal = &r.Objectives.Entries[i]
				}
			}
			eligible := tier == content.MobEliteMinion || tier == content.MobElite || tier == content.MobMiniboss
			if !eligible {
				if goal != nil {
					t.Fatal("non-elite generated objective")
				}
				continue
			}
			if goal == nil {
				t.Fatal("ranged elite omitted")
			}
			if wrong {
				r.hurtEnemy(1, 1000, "hit")
				r.hurtEnemy(0, 1000, "hit")
			} else {
				r.hurtEnemy(0, 1000, "hit")
				r.hurtEnemy(1, 1000, "hit")
			}
			r.UpdateObjectives()
			if (goal.Status == "failed") != wrong {
				t.Fatalf("wrong priority for %s", tier)
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
			if wrong {
				want = 0
			}
			if saved.ObjectiveHistory["Wayfarer"]["elite_priority"] != want {
				t.Fatal("wrong elite banking")
			}
		}
	}
}

func TestElitePriorityEligibilityAcrossCampaign(t *testing.T) {
	for level := 1; level <= LevelCount; level++ {
		r := NewRunAtLevel("elite-eligibility", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level)
		planned, offered := false, false
		for _, room := range r.EncounterPlan {
			for _, enemy := range room {
				planned = planned || isPriorityElite(enemy)
			}
		}
		for _, entry := range r.Objectives.Entries {
			offered = offered || entry.ID == "elite_priority"
		}
		if planned != offered {
			t.Fatalf("wrong eligibility at mission %d", level)
		}
	}
	if !isPriorityElite(Actor{Kind: "knight"}) || isPriorityElite(Actor{Kind: "knight", Tier: string(content.MobCommon)}) {
		t.Fatal("legacy knight fallback overrides canonical tier")
	}
}
