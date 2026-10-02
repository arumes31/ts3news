package rift

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestClassPracticeUsesEquippedBuildAndRequiresChargedHit(t *testing.T) {
	build := testRun().Build
	build.Signatures = []Skill{{ID: "current-builder", Name: "Current builder", Role: "builder", Kind: "fire", Power: 1}, {ID: "current-finisher", Name: "Current finisher", Role: "finisher", Kind: "fire", Power: 1}}
	r, err := NewPracticeRun("class", build, "class", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.Build.Signatures, build.Signatures) || !reflect.DeepEqual(r.Build.Skills, build.Skills) {
		t.Fatal("practice replaced equipped abilities")
	}
	if r.practiceInput(Input{Skill: "current-builder"}).Skill != "current-builder" {
		t.Fatal("class skills filtered out")
	}
	r.Player.X = 1000
	r.practiceTick()
	if r.Practice.Completed {
		t.Fatal("walking completed class drill")
	}
	r.skillHit(0, 10, build.Signatures[1], 0, "")
	r.practiceTick()
	if r.Practice.Completed {
		t.Fatal("empty finisher completed drill")
	}
	r.Resource = 1
	charges, mark := r.classCast(build.Signatures[1])
	r.practiceTick()
	if r.Practice.Completed {
		t.Fatal("missed finisher completed drill")
	}
	r.skillHit(0, 10, build.Signatures[1], charges, mark)
	r.practiceTick()
	if !r.Practice.Completed || r.Status != "complete" || r.Practice.ClassHits != 1 || len(r.Drops) != 0 || r.Gold != 0 || len(r.History) != 0 {
		t.Fatal("invalid class drill completion or rewards")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Practice.ClassHits != 1 {
		t.Fatal("progress lost in save")
	}
	if err = saved.ResetPractice(time.Unix(110, 0)); err != nil {
		t.Fatal(err)
	}
	if saved.Practice.ClassHits != 0 || saved.Practice.Completed || !reflect.DeepEqual(saved.Build.Signatures, build.Signatures) {
		t.Fatal("reset lost build or retained progress")
	}
	if _, err = NewPracticeRun("invalid", Build{HP: 100}, "class", time.Now()); err == nil {
		t.Fatal("missing class signatures accepted")
	}
}
