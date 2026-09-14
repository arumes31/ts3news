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
