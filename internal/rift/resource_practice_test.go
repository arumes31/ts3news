package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func TestResourcePracticeRequiresTwoFullChargeCycles(t *testing.T) {
	build := testRun().Build
	build.Signatures = []Skill{{ID: "builder", Role: "builder", Kind: "fire", Power: 1}, {ID: "finisher", Role: "finisher", Kind: "fire", Power: 1}}
	r, err := NewPracticeRun("resource", build, "resource", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	r.Player.X = 1000
	r.practiceTick()
	if r.Practice.Completed {
		t.Fatal("walking completed drill")
	}
	cast := func(index int) {
		t.Helper()
		if err := r.PracticeTool("practice_cooldowns"); err != nil {
			t.Fatal(err)
		}
		r.Player.Cooldown = 0 // Advance past the shared cast lock for this engine exercise.
		r.cast(build.Signatures[index].ID)
	}
	cycle := func() { cast(0); cast(0); cast(0); cast(1); r.practiceTick() }
	cycle()
	if r.Practice.ResourceCycles != 1 || r.Practice.Completed {
		t.Fatal("first full cycle not recorded")
	}
	r.Player.Cooldown = 1
	r.cast(build.Signatures[1].ID)
	if r.Practice.ResourceCycles != 1 {
		t.Fatal("blocked cast changed progress")
	}
	cast(1)
	if r.Practice.ResourceCycles != 0 {
		t.Fatal("empty finisher did not reset streak")
	}
	cycle()
	cast(0)
	cast(1)
	if r.Practice.ResourceCycles != 0 {
		t.Fatal("partial finisher did not reset streak")
	}
	cycle()
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err = json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	r = &restored
	if r.Practice.ResourceCycles != 1 {
		t.Fatal("cycle lost on save")
	}
	cycle()
	if !r.Practice.Completed || r.Status != "complete" || r.Practice.ResourceCycles != 2 {
		t.Fatal("two full cycles did not complete drill")
	}
	if r.Gold != 0 || len(r.Drops) != 0 || len(r.History) != 0 {
		t.Fatal("practice generated rewards")
	}
	if err = r.ResetPractice(time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	if r.Practice.ResourceCycles != 0 || r.Resource != 0 || r.Practice.Completed {
		t.Fatal("reset retained progress")
	}
	if _, err = NewPracticeRun("invalid", Build{HP: 100}, "resource", time.Now()); err == nil {
		t.Fatal("accepted missing signatures")
	}
}
