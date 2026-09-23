package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestObjectiveHistoryBanksOnceByFrozenDifficulty(t *testing.T) {
	r := NewRunAtLevel("bank-objectives", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
	r.Room = 2
	r.Status = "cleared"
	r.Stats.Guards = 5
	r.UpdateObjectives()
	if len(r.ObjectiveHistory) != 0 {
		t.Fatal("unbanked objectives counted")
	}
	r.FinishCheckpoint("bank", nil)
	if !r.Objectives.Banked || r.ObjectiveHistory["Wayfarer"]["guard"] != 1 {
		t.Fatal("banked completion missing")
	}
	r.FinishCheckpoint("bank", nil)
	if r.ObjectiveHistory["Wayfarer"]["guard"] != 1 {
		t.Fatal("repeat bank counted twice")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	next := NewRunAtLevel("next-objectives", r.Build, time.Unix(200, 0), content.AbyssMobCatalog(), 26)
	next.InheritCampaignHistory(&saved)
	next.Room = 2
	next.Status = "cleared"
	next.Stats.Guards = 5
	next.FinishCheckpoint("bank", nil)
	if next.ObjectiveHistory["Veteran"]["guard"] != 1 || next.ObjectiveHistory["Wayfarer"]["guard"] != 1 {
		t.Fatal("difficulty records merged incorrectly")
	}
	next.ObjectiveHistory["Wayfarer"]["guard"] = 9
	if saved.ObjectiveHistory["Wayfarer"]["guard"] != 1 {
		t.Fatal("new expedition mutated previous history")
	}
}

func TestObjectiveHistoryExcludesFailedAndUnfinishedGoals(t *testing.T) {
	r := NewRunAtLevel("failed-goals", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
	r.Stats.DamageTaken = 1
	r.Room = 2
	r.Status = "cleared"
	r.FinishCheckpoint("bank", nil)
	if r.ObjectiveHistory["Wayfarer"]["no_damage"] != 0 || r.ObjectiveHistory["Wayfarer"]["guard"] != 0 || r.ObjectiveHistory["Wayfarer"]["timed"] != 1 {
		t.Fatal("failed objectives counted")
	}
	early := NewRunAtLevel("early", r.Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
	early.Status = "cleared"
	early.FinishCheckpoint("exit", nil)
	if len(early.ObjectiveHistory) != 0 {
		t.Fatal("early exit counted objectives")
	}
}
