package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestMissionObjectivesTrackFailuresAndCompletion(t *testing.T) {
	r := NewRunAtLevel("objectives", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
	if r.Objectives == nil || len(r.Objectives.Entries) != 4 {
		t.Fatal("objectives missing")
	}
	r.Stats.Seconds = 181
	r.Stats.DamageTaken = 1
	r.Stats.SkillsCast = 1
	r.Stats.Guards = 5
	r.UpdateObjectives()
	for i := 0; i < 3; i++ {
		if r.Objectives.Entries[i].Status != "failed" || r.Objectives.Entries[i].Reason == "" {
			t.Fatal("objective violation not explained")
		}
	}
	if r.Status != "fighting" || r.Objectives.Entries[3].Status != "active" {
		t.Fatal("optional failure ended run or guard completed early")
	}
	r.Room = 2
	r.Status = "cleared"
	r.UpdateObjectives()
	if r.Objectives.Entries[3].Status != "complete" || !r.Objectives.Finished {
		t.Fatal("guard objective did not complete with mission")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	saved.Stats.Guards = 0
	saved.UpdateObjectives()
	if saved.Objectives.Entries[3].Status != "complete" {
		t.Fatal("saved result changed after completion")
	}
}

func TestObjectivesUseMissionBaselineAndPreserveRetryFailures(t *testing.T) {
	r := NewRunAtLevel("baseline", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
	r.Stats.Seconds = 200
	r.Stats.DamageTaken = 40
	r.Stats.SkillsCast = 4
	r.Stats.Guards = 10
	r.setLevel(2, content.AbyssMobCatalog())
	r.UpdateObjectives()
	for _, entry := range r.Objectives.Entries {
		if entry.Current != 0 || entry.Status != "active" {
			t.Fatal("previous mission polluted objectives")
		}
	}
	r.Room = 2
	r.Player.HP = 0
	r.tick(Input{}, .02)
	if !r.Objectives.Finished {
		t.Fatal("defeat left objectives pending")
	}
	if err := r.RetryBossEncounter(time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	r.UpdateObjectives()
	for _, entry := range r.Objectives.Entries {
		if entry.Status != "failed" {
			t.Fatal("retry erased optional failure")
		}
	}
}

func TestObjectiveTimeIncludesBoundaryAndExcludesPausedTime(t *testing.T) {
	r := NewRunAtLevel("boundary", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
	r.SetPaused(true, time.Unix(100, 0))
	r.Step(Input{}, time.Unix(200, 0))
	if r.Objectives.Entries[0].Current != 0 {
		t.Fatal("pause advanced objective clock")
	}
	r.Stats.Seconds = 180
	r.Stats.Guards = 5
	r.Room = 2
	r.Status = "cleared"
	r.UpdateObjectives()
	for _, entry := range r.Objectives.Entries {
		if entry.Status != "complete" {
			t.Fatalf("valid mission failed %s", entry.ID)
		}
	}
}

func TestObjectivesFollowConfirmedCombatNotRejectedInputs(t *testing.T) {
	r := NewRunAtLevel("actions", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
	r.Player.Mana = 0
	r.tick(Input{Skill: "fire"}, .02)
	if r.Objectives.Entries[2].Status != "active" {
		t.Fatal("rejected ability failed objective")
	}
	r.Player.Mana = 100
	r.tick(Input{Skill: "fire"}, .02)
	if r.Objectives.Entries[2].Status != "failed" {
		t.Fatal("confirmed ability did not fail objective")
	}
	r.hurtPlayer(20, r.Player.X+30, r.Player.Y)
	r.tick(Input{}, .02)
	if r.Objectives.Entries[1].Status != "failed" || r.Status != "fighting" {
		t.Fatal("damage objective did not fail independently")
	}
}

func TestObjectiveResultsSurviveSeamlessAdvanceAndNewExpedition(t *testing.T) {
	r := NewRunAtLevel("results", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
	r.Room = 2
	r.Status = "cleared"
	r.Stats.Guards = 5
	r.Stats.Seconds = 42
	r.FinishCheckpoint("advance", content.AbyssMobCatalog())
	if r.LastObjectives == nil || !r.LastObjectives.Finished || r.LastObjectives.Mission != 1 || r.Objectives.Mission != 2 {
		t.Fatal("advancement discarded results")
	}
	for _, entry := range r.LastObjectives.Entries {
		if entry.Status != "complete" {
			t.Fatal("checkpoint did not finalize objective")
		}
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	next := NewRunAtLevel("new-results", r.Build, time.Unix(200, 0), content.AbyssMobCatalog(), 3)
	next.InheritCampaignHistory(&saved)
	if next.LastObjectives == nil || next.LastObjectives.Mission != 1 {
		t.Fatal("new expedition lost last results")
	}
	next.Stats.SkillsCast = 1
	next.UpdateObjectives()
	if saved.LastObjectives.Entries[2].Status != "complete" {
		t.Fatal("current progress mutated prior result")
	}
}
