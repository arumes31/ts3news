package rift

import (
	"testing"
	"time"
)

func TestPracticeMovementUsesControlsWithoutRewardsOrCampaignHistory(t *testing.T) {
	r, err := NewPracticeRun("practice", testRun().Build, "movement", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 200 && r.Status == "fighting"; i++ {
		r.Step(Input{X: 1, Attack: true, Jump: true, Skill: "fire"}, time.UnixMilli(100000+int64(i)*50))
	}
	if r.Status != "complete" || !r.Practice.Completed || r.Stats.Attacks != 0 || r.Stats.Jumps != 0 || len(r.Drops) != 0 || len(r.History) != 0 || len(r.CompletedLevels) != 0 {
		t.Fatalf("invalid movement practice: %+v", r)
	}
	r.Status = "cleared"
	r.FinishCheckpoint("advance", nil)
	if r.NextRoom() || len(r.History) != 0 || r.Room != 0 {
		t.Fatal("practice entered campaign progression")
	}
	if _, err := NewPracticeRun("bad", r.Build, "unknown", time.Now()); err == nil {
		t.Fatal("unknown drill accepted")
	}
}

func TestPracticeJumpRequiresCrossingCoverAndComboRequiresHits(t *testing.T) {
	r, _ := NewPracticeRun("jump", testRun().Build, "jump", time.Unix(100, 0))
	for i := 1; i <= 120; i++ {
		r.Step(Input{X: 1}, time.UnixMilli(100000+int64(i)*50))
	}
	if r.Status != "fighting" || r.Player.X >= 450 {
		t.Fatal("walk bypassed jump lane")
	}
	for i := 121; i <= 350 && r.Status == "fighting"; i++ {
		r.Step(Input{X: 1, Jump: true}, time.UnixMilli(100000+int64(i)*50))
	}
	if !r.Practice.Completed {
		t.Fatal("jump did not cross practice cover")
	}
	combo, _ := NewPracticeRun("combo", testRun().Build, "combo", time.Unix(100, 0))
	combo.Player.Facing = -1
	for i := 1; i <= 30; i++ {
		combo.Step(Input{Attack: true}, time.UnixMilli(100000+int64(i)*50))
	}
	if combo.Practice.Completed || combo.Practice.Hits != 0 {
		t.Fatal("missed attacks completed drill")
	}
	combo.Player.Facing = 1
	for i := 31; i <= 100 && combo.Status == "fighting"; i++ {
		combo.Step(Input{Attack: true}, time.UnixMilli(100000+int64(i)*50))
	}
	if !combo.Practice.Completed || combo.Practice.Hits < 3 || len(combo.Drops) != 0 {
		t.Fatal("basic combo target did not complete safely")
	}
	if err := combo.ResetPractice(time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	if combo.Status != "fighting" || combo.Practice.Hits != 0 || combo.Stats.Attacks != 0 || combo.ID != "combo" {
		t.Fatal("practice reset failed")
	}
}

func TestPracticeDirectionalGuardRequiresFacingAttacker(t *testing.T) {
	r, err := NewPracticeRun("guard", testRun().Build, "guard", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	r.Player.Facing = -1
	for i := 1; i <= 100; i++ {
		r.Step(Input{Guard: true, Attack: true, Jump: true, Skill: "fire"}, time.UnixMilli(100000+int64(i)*50))
	}
	if r.Stats.Guards != 0 || r.Practice.Completed || r.Stats.DamageTaken == 0 {
		t.Fatal("rear attacks counted as guarded")
	}
	r.Player.Facing = 1
	for i := 101; i <= 300 && r.Status == "fighting"; i++ {
		r.Step(Input{Guard: true}, time.UnixMilli(100000+int64(i)*50))
	}
	if !r.Practice.Completed || r.Stats.Guards < 3 || r.Stats.Attacks != 0 || r.Stats.Jumps != 0 || len(r.Drops) != 0 || len(r.History) != 0 {
		t.Fatalf("guard drill did not complete safely: %+v", r.Stats)
	}
}

func TestPracticeHazardRequiresAvoidingThreeRetargetedPulses(t *testing.T) {
	idle, err := NewPracticeRun("idle", testRun().Build, "hazard", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 190 && idle.Status == "fighting"; i++ {
		idle.Step(Input{}, time.UnixMilli(100000+int64(i)*50))
	}
	if idle.Stats.DamageTaken == 0 || idle.Practice.Completed || idle.Practice.Dodges != 0 {
		t.Fatal("standing in hazards passed the drill")
	}
	r, _ := NewPracticeRun("dodge", testRun().Build, "hazard", time.Unix(100, 0))
	for i := 1; i <= 300 && r.Status == "fighting"; i++ {
		h := r.Arena().Hazards[0]
		in := Input{Attack: true, Skill: "fire"}
		if h.Phase(r.Clock) < 1.2 && contains(h.Obstacle, r.Player.X, r.Player.Y, 12) {
			if r.Player.X < Width/2 {
				in.X = 1
			} else {
				in.X = -1
			}
		}
		r.Step(in, time.UnixMilli(100000+int64(i)*50))
	}
	if !r.Practice.Completed || r.Practice.Dodges != 3 || r.Stats.DamageTaken != 0 || r.Stats.Attacks != 0 || len(r.Drops) != 0 {
		t.Fatalf("hazard drill did not complete: %+v", r.Practice)
	}
	if r.Arena().Hazards[0].X == 110 {
		t.Fatal("hazard did not retarget")
	}
}

func TestPracticeHazardCanBeJumpedDuringTheWarning(t *testing.T) {
	r, _ := NewPracticeRun("jump-warning", testRun().Build, "hazard", time.Unix(100, 0))
	for i := 1; i <= 300 && r.Status == "fighting"; i++ {
		phase := r.Arena().Hazards[0].Phase(r.Clock)
		r.Step(Input{Jump: phase > 1.1 && phase < 1.2}, time.UnixMilli(100000+int64(i)*50))
	}
	if !r.Practice.Completed || r.Stats.DamageTaken != 0 || r.Stats.Jumps != 3 {
		t.Fatalf("timed jumps failed: %+v", r.Stats)
	}
}

func TestPracticeHazardDamageResetsEarnedStreak(t *testing.T) {
	r, _ := NewPracticeRun("streak", testRun().Build, "hazard", time.Unix(100, 0))
	for i := 1; i <= 40; i++ {
		r.Step(Input{X: 1}, time.UnixMilli(100000+int64(i)*50))
	}
	if r.Practice.Dodges != 1 {
		t.Fatal("first clean pulse was not recorded")
	}
	for i := 41; i <= 110; i++ {
		r.Step(Input{}, time.UnixMilli(100000+int64(i)*50))
	}
	if r.Practice.Dodges != 0 || r.Stats.DamageTaken == 0 {
		t.Fatal("missed warning retained the clean streak")
	}
}
