package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPerfectGuardCareerCountsConfirmedCampaignGuards(t *testing.T) {
	r := testRun()
	r.Player.Guard = true
	r.Player.Facing = 1
	r.SkillTimers["perfect_guard"] = .22
	r.hurtPlayer(1, r.Player.X+20, r.Player.Y)
	if r.Stats.PerfectGuards != 1 {
		t.Fatal("perfect guard not counted")
	}
	r.hurtPlayer(1, r.Player.X-20, r.Player.Y)
	r.SkillTimers["perfect_guard"] = 0
	r.hurtPlayer(1, r.Player.X+20, r.Player.Y)
	if r.Stats.PerfectGuards != 1 {
		t.Fatal("rear or ordinary guard counted")
	}
	r.PastExpeditions.PerfectGuards = 9
	raw, _ := json.Marshal(r)
	var saved Run
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	next := NewRunAtLevel("new", r.Build, time.Now(), nil, 1)
	next.InheritCampaignHistory(&saved)
	if next.RecordedTotals().PerfectGuards != 10 {
		t.Fatal("career guards not inherited")
	}
	if next.Gold != 0 || len(next.Drops) != 0 {
		t.Fatal("badge created rewards")
	}
	r.Practice = &PracticeState{Mode: "guard"}
	r.SkillTimers["perfect_guard"] = .22
	r.hurtPlayer(1, r.Player.X+20, r.Player.Y)
	if r.Stats.PerfectGuards != 1 {
		t.Fatal("practice guard counted")
	}
}
