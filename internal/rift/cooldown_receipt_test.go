package rift

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestChronomancerReceiptRecordsActualCooldownRecovery(t *testing.T) {
	r := testRun()
	r.Build.Class = "chronomancer"
	r.Resource = 2
	finisher := Skill{ID: "finish", Name: "Temporal Release", Role: "finisher"}
	r.Build.Signatures = []Skill{{ID: "build", Name: "Time Bolt"}, finisher}
	r.Build.Skills = []Skill{{ID: "fire", Name: "Fireball"}, {ID: "ready", Name: "Ready skill"}}
	r.Build.Ultimate = nil
	r.SkillTimers = map[string]float64{"finish": 8, "build": .4, "fire": 5, "ready": 0, "jump": .7, "hazard-hit": .3}
	r.classCast(finisher)
	want := []CooldownRecovery{{Name: "Time Bolt", Seconds: .4}, {Name: "Fireball", Seconds: 1.5}, {Name: "Jump", Seconds: .7}}
	if r.LastCooldownReceipt.Source != finisher.Name || !reflect.DeepEqual(r.LastCooldownReceipt.Recovered, want) {
		t.Fatalf("wrong receipt: %+v", r.LastCooldownReceipt)
	}
	if r.SkillTimers["fire"] != 3.5 || r.SkillTimers["finish"] != 8 || r.SkillTimers["hazard-hit"] != .3 {
		t.Fatal("wrong cooldowns reduced")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved.LastCooldownReceipt, r.LastCooldownReceipt) {
		t.Fatal("receipt lost after save")
	}
	saved.classCast(finisher)
	if !reflect.DeepEqual(saved.LastCooldownReceipt, r.LastCooldownReceipt) {
		t.Fatal("empty cast replaced receipt")
	}
	saved.Resource = 1
	saved.SkillTimers["fire"] = 0
	saved.classCast(finisher)
	if len(saved.LastCooldownReceipt.Recovered) != 0 {
		t.Fatal("ready cooldown included in receipt")
	}
}
