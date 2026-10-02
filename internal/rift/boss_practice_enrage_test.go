package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func enrageTestRun(t *testing.T) *Run {
	t.Helper()
	r, err := NewPracticeRun("enrage", testRun().Build, "boss", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	r.Practice.EnrageSeconds = BossPracticeEnrageSeconds
	r.SkillTimers = map[string]float64{}
	r.Build.Armor = 0
	r.FirstHitGrace = false
	r.Player.HP = 10000
	r.Player.MaxHP = 10000
	return r
}

func TestBossPracticeEnrageImpactBoundaryAndOwnership(t *testing.T) {
	for _, tc := range []struct {
		name    string
		clock   float64
		enabled bool
		owner   bool
		want    float64
	}{
		{"before", 29.999, true, true, 20}, {"at threshold", 30, true, true, 30}, {"after", 50, true, true, 30}, {"ordinary", 50, false, true, 20}, {"other owner", 50, true, false, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := enrageTestRun(t)
			r.Clock = tc.clock
			if !tc.enabled {
				r.Practice.EnrageSeconds = 0
			}
			owner := r.Practice.BossStart.ID
			if !tc.owner {
				owner = "other"
			}
			before := r.Player.HP
			r.hurtPlayerFromEnemyGuardable(20, r.Player.X+100, r.Player.Y, owner, false)
			if before-r.Player.HP != tc.want {
				t.Fatalf("impact damage=%v want%v", before-r.Player.HP, tc.want)
			}
			before = r.Player.HP
			r.hurtPlayerFromHazard(20, r.Player.X, r.Player.Y)
			if before-r.Player.HP != 20 {
				t.Fatal("enrage amplified hazard damage")
			}
		})
	}
	r := enrageTestRun(t)
	r.Clock = 40
	r.Practice = nil
	if r.bossPracticeDamage(20, "practice-boss") != 20 {
		t.Fatal("campaign damage amplified")
	}
}

func TestBossPracticeEnragePauseSaveCueAndReset(t *testing.T) {
	r := enrageTestRun(t)
	r.Clock = 29.5
	r.SetPaused(true, time.UnixMilli(r.LastMS))
	r.Step(Input{}, time.UnixMilli(r.LastMS+60000))
	if r.Clock != 29.5 || r.Practice.EnrageTriggered {
		t.Fatal("pause advanced enrage")
	}
	r.SetPaused(false, time.UnixMilli(r.LastMS))
	r.Clock = 30
	for i := 0; i < 5; i++ {
		r.practiceTick()
	}
	count := 0
	for _, event := range r.Events {
		if event.Kind == "practice_enrage" {
			count++
		}
	}
	if count != 1 || !r.Practice.EnrageTriggered {
		t.Fatalf("enrage cues=%d", count)
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	before := len(saved.Events)
	saved.practiceTick()
	if len(saved.Events) != before || saved.bossPracticeDamage(20, saved.Practice.BossStart.ID) != 30 {
		t.Fatal("saved enrage changed")
	}
	if err = saved.ResetPractice(time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	if saved.Clock != 0 || saved.Practice.EnrageTriggered || saved.Practice.EnrageSeconds != 30 || saved.bossPracticeDamage(20, saved.Practice.BossStart.ID) != 20 {
		t.Fatal("reset did not restart challenge")
	}
	if saved.Gold != 0 || len(saved.Drops) != 0 {
		t.Fatal("practice granted rewards")
	}
}
