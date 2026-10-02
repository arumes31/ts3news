package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestUltimateSavingObjective(t *testing.T) {
	for _, equipped := range []bool{false, true} {
		build := testRun().Build
		if equipped {
			build.Ultimate = &Skill{ID: "ult", Kind: "ultimate", Cost: 20, Power: 1}
		}
		r := NewRunAtLevel("ultimate-goal", build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
		var goal *ObjectiveProgress
		for i := range r.Objectives.Entries {
			if r.Objectives.Entries[i].ID == "save_ultimate" {
				goal = &r.Objectives.Entries[i]
			}
		}
		if !equipped {
			if goal != nil {
				t.Fatal("unequipped ultimate offered")
			}
			continue
		}
		if goal == nil {
			t.Fatal("ultimate objective missing")
		}
		r.Player.Mana = 0
		r.cast("ult")
		r.UpdateObjectives()
		if goal.Status != "active" {
			t.Fatal("rejected cast failed objective")
		}
		r.Player.Mana = 100
		r.cast("fire")
		r.UpdateObjectives()
		if goal.Status != "active" {
			t.Fatal("ordinary ability failed objective")
		}
		r.Player.Cooldown = 0
		r.cast("ult")
		r.UpdateObjectives()
		if goal.Status != "failed" || goal.Current != 1 || r.Status != "fighting" {
			t.Fatal("confirmed ultimate not tracked independently")
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
		if saved.ObjectiveHistory["Wayfarer"]["save_ultimate"] != 1 {
			t.Fatal("previous mission ultimate polluted new mission or completion not banked")
		}
	}
}
