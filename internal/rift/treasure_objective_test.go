package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestTreasureObjectiveEligibilityAndCapture(t *testing.T) {
	for _, outcome := range []string{"absent", "escape", "capture"} {
		t.Run(outcome, func(t *testing.T) {
			r := NewRunAtLevel("treasure-goal", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
			r.EncounterPlan = [][]Actor{{{ID: "t", Kind: "treasure", HP: 1, MaxHP: 1, X: 500, Y: 400}}}
			if outcome == "absent" {
				r.EncounterPlan[0][0].Kind = "goblin"
			}
			r.Stats.TreasureGoblins = 3
			r.beginObjectives()
			var goal *ObjectiveProgress
			for i := range r.Objectives.Entries {
				if r.Objectives.Entries[i].ID == "treasure_capture" {
					goal = &r.Objectives.Entries[i]
				}
			}
			if outcome == "absent" {
				if goal != nil {
					t.Fatal("impossible capture offered")
				}
				return
			}
			if goal == nil {
				t.Fatal("capture objective missing")
			}
			r.Enemies = append([]Actor{}, r.EncounterPlan[0]...)
			if outcome == "escape" {
				r.EscapeEnemy(0)
			} else {
				r.hurtEnemy(0, 100, "hit")
			}
			r.UpdateObjectives()
			if goal.Status != "active" {
				t.Fatal("goal ended before mission clear")
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
			want := 0
			if outcome == "capture" {
				want = 1
			}
			if saved.ObjectiveHistory["Wayfarer"]["treasure_capture"] != want {
				t.Fatal("capture history counted escape or missed kill")
			}
			for _, entry := range saved.Objectives.Entries {
				if entry.ID == "treasure_capture" {
					if int(entry.Current) != want || (entry.Status == "complete") != (want == 1) {
						t.Fatalf("wrong result: %+v", entry)
					}
				}
			}
		})
	}
}

func TestTreasureObjectiveMatchesEveryMissionPlan(t *testing.T) {
	for level := 1; level <= LevelCount; level++ {
		r := NewRunAtLevel("treasure-campaign", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level)
		planned, offered := false, false
		for _, room := range r.EncounterPlan {
			for _, enemy := range room {
				planned = planned || enemy.Kind == "treasure"
			}
		}
		for _, entry := range r.Objectives.Entries {
			offered = offered || entry.ID == "treasure_capture"
		}
		if planned != offered {
			t.Fatalf("mission %d objective does not match frozen plan", level)
		}
	}
}
