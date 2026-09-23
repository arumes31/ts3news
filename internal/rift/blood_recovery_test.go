package rift

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestBloodRecoveryReceiptsIncludeActualHealingAndOverflow(t *testing.T) {
	r := testRun()
	r.Build.Class = "bloodblade"
	r.Player.MaxHP = 100
	r.Player.HP = 95
	r.Build.Signatures = []Skill{{ID: "leech", Name: "Leech Cut", Role: "builder", Kind: "slash", Heal: .04}, {ID: "reap", Name: "Crimson Reap", Role: "finisher", Kind: "slash"}}
	r.cast("leech")
	if r.LastBloodRecovery == nil || r.LastBloodRecovery.SkillName != "Leech Cut" || r.LastBloodRecovery.Healed != 4 || r.LastBloodRecovery.Overflow != 0 {
		t.Fatalf("builder receipt: %+v", r.LastBloodRecovery)
	}
	r.Player.Cooldown = 0
	r.Resource = 3
	r.cast("reap")
	want := &BloodRecovery{SkillID: "reap", SkillName: "Crimson Reap", Healed: 1, Overflow: 11}
	if !reflect.DeepEqual(r.LastBloodRecovery, want) {
		t.Fatalf("finisher receipt: %+v", r.LastBloodRecovery)
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved.LastBloodRecovery, want) {
		t.Fatal("receipt lost in save")
	}
	saved.healPlayerBySkill(4, "leech")
	if saved.LastBloodRecovery.Healed != 0 || saved.LastBloodRecovery.Overflow != 4 {
		t.Fatal("full-health receipt incorrect")
	}
	prior := saved.LastBloodRecovery
	saved.healPlayer(5)
	saved.Build.Class = "vanguard"
	saved.healPlayerBySkill(5, "leech")
	if saved.LastBloodRecovery != prior {
		t.Fatal("unrelated healing replaced class receipt")
	}
}
