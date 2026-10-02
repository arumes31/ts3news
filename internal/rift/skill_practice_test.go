package rift

import (
	"reflect"
	"testing"
	"time"
)

func TestSkillPracticeAcceptsFoundationBuildAndStaysOpen(t *testing.T) {
	build := testRun().Build
	build.Signatures = nil
	build.Skills = []Skill{{ID: "heal-test", Name: "Healing", Kind: "heal", Power: 1, Cost: 10}}
	r, err := NewPracticeRun("sandbox", build, "skills", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.Build.Skills, build.Skills) || len(r.Build.Signatures) != 0 {
		t.Fatal("changed equipped abilities")
	}
	if r.Player.HP != r.Player.MaxHP*.6 {
		t.Fatal("no missing health for healing")
	}
	input := Input{Skill: "heal-test", Attack: true, Jump: true, Guard: true, X: 1, Y: 1}
	if r.practiceInput(input) != input {
		t.Fatal("sandbox filters abilities")
	}
	r.hurtEnemy(0, 10000000, "hit")
	if r.Enemies[0].HP != r.Enemies[0].MaxHP {
		t.Fatal("target can die")
	}
	r.Player.X = 1000
	r.practiceTick()
	if r.Status != "fighting" || r.Practice.Completed || r.Gold != 0 || len(r.Drops) != 0 {
		t.Fatal("sandbox completed or earned rewards")
	}
	if err = r.PracticeTool("practice_mana"); err != nil {
		t.Fatal(err)
	}
	if err = r.ResetPractice(time.Unix(110, 0)); err != nil {
		t.Fatal(err)
	}
	if r.Practice.Mode != "skills" || r.Player.HP != r.Player.MaxHP*.6 || r.Practice.Hits != 0 {
		t.Fatal("reset did not restore sandbox")
	}
}
