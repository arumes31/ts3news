package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestBossPracticePhasesResetWithoutRewards(t *testing.T) {
	var name string
	for _, mob := range content.AbyssMobCatalog() {
		if AdaptMonster(mob).Kind == "boss" {
			name = mob.Name
			break
		}
	}
	for phase := 1; phase <= 3; phase++ {
		r, err := NewBossPracticeRun("drill", testRun().Build, name, phase, time.Unix(100, 0))
		if err != nil {
			t.Fatal(err)
		}
		start := r.Enemies[0]
		if start.Name != name || start.Phase != phase || start.HP != start.MaxHP*float64(bossPhaseTraining[phase-1].AtHealthPercent)/100 {
			t.Fatal("wrong starting boss phase")
		}
		r.Revision = 7
		r.Epoch = "economy"
		r.StartKey = "request"
		r.Player.X = 900
		r.practiceTick()
		if r.Practice.Completed {
			t.Fatal("walking completed boss practice")
		}
		input := Input{X: 1, Jump: true, Attack: true, Skill: "fire", Guard: true}
		if r.practiceInput(input) != input {
			t.Fatal("boss practice filtered combat controls")
		}
		r.hurtEnemy(0, 1, "fire")
		if r.Enemies[0].HP >= start.HP {
			t.Fatal("boss practice target healed damage")
		}
		r.hurtEnemy(0, start.MaxHP*100, "fire")
		r.practiceTick()
		if !r.Practice.Completed || r.Status != "complete" {
			t.Fatal("defeated boss did not complete drill")
		}
		if r.Gold != 0 || len(r.Drops) != 0 || len(r.History) != 0 || len(r.MonsterRecords) != 0 || r.Stats.Kills != 0 {
			t.Fatal("boss practice earned campaign rewards or records")
		}
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved Run
		if err = json.Unmarshal(data, &saved); err != nil {
			t.Fatal(err)
		}
		if err = saved.ResetPractice(time.Unix(200, 0)); err != nil {
			t.Fatal(err)
		}
		if saved.Enemies[0] != start || saved.Practice.Completed || saved.Status != "fighting" || saved.Revision != 7 || saved.Epoch != "economy" || saved.StartKey != "request" {
			t.Fatal("reset did not restore saved boss phase")
		}
	}
}

func TestBossPracticeRejectsInvalidSelection(t *testing.T) {
	for _, phase := range []int{0, 4} {
		if _, err := NewBossPracticeRun("bad", Build{}, "unknown", phase, time.Now()); err == nil {
			t.Fatal("invalid phase accepted")
		}
	}
	if _, err := NewBossPracticeRun("bad", Build{}, "unknown", 1, time.Now()); err == nil {
		t.Fatal("unknown boss accepted")
	}
	for _, mob := range content.AbyssMobCatalog() {
		if AdaptMonster(mob).Kind != "boss" {
			if _, err := NewBossPracticeRun("bad", Build{}, mob.Name, 1, time.Now()); err == nil {
				t.Fatal("ordinary monster accepted")
			}
			break
		}
	}
}
