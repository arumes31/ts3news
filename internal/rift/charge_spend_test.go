package rift

import (
	"encoding/json"
	"testing"
)

func TestChargeSpendNamesConfirmedConsumer(t *testing.T) {
	r := circleTestRun()
	first := Skill{ID: "first", Name: "First finisher", Role: "finisher"}
	r.Resource = 3
	r.classCast(first)
	if r.LastChargeSpend == nil || *r.LastChargeSpend != (ChargeSpend{SkillID: first.ID, SkillName: first.Name, Charges: 3}) {
		t.Fatal("missing spending identity")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if *saved.LastChargeSpend != *r.LastChargeSpend {
		t.Fatal("spending identity lost in save")
	}
	saved.classCast(Skill{ID: "empty", Name: "Empty cast", Role: "finisher"})
	if saved.LastChargeSpend.SkillID != "first" {
		t.Fatal("empty finisher replaced spend")
	}
	saved.classCast(Skill{Role: "builder"})
	if saved.LastChargeSpend.SkillID != "first" {
		t.Fatal("builder replaced spend")
	}
	saved.classCast(Skill{ID: "second", Name: "Second finisher", Role: "finisher"})
	if saved.LastChargeSpend.SkillID != "second" || saved.LastChargeSpend.Charges != 1 {
		t.Fatal("latest spend not updated")
	}
}
