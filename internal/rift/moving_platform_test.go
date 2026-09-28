package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func movingPlatformRun(t *testing.T) *Run {
	t.Helper()
	r, err := NewPracticeRun("ferry", testRun().Build, "moving_platform", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestMovingPlatformCarriesGroundedRidersAndLeavesJumpers(t *testing.T) {
	for _, jump := range []float64{0, .8} {
		r := movingPlatformRun(t)
		p := r.Arena().Platforms[0]
		r.Player.X, r.Player.Y, r.Player.Jump = p.X+p.W/2, p.Y+p.H/2, jump
		before := r.Player.X
		r.Clock = 1
		r.practiceTick()
		moved := r.Arena().Platforms[0].X - p.X
		if moved <= 0 {
			t.Fatal("deck did not move")
		}
		if jump == 0 {
			if math.Abs(r.Player.X-before-moved) > 1e-8 || r.Player.Elevation != p.Rise || r.Practice.PlatformRide != moved {
				t.Fatal("grounded rider lost deck or credit")
			}
		} else if r.Player.X != before || r.Practice.PlatformRide != 0 {
			t.Fatal("airborne rider carried or credited")
		}
	}
}

func TestMovingPlatformPauseSaveAndReset(t *testing.T) {
	r := movingPlatformRun(t)
	p := r.Arena().Platforms[0]
	r.Player.X, r.Player.Y = p.X+p.W/2, p.Y+p.H/2
	for i := 0; i < 40; i++ {
		r.Step(Input{}, time.UnixMilli(r.LastMS+50))
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30; i++ {
		now := time.UnixMilli(r.LastMS + 50)
		r.Step(Input{}, now)
		saved.Step(Input{}, now)
		if r.Player.X != saved.Player.X || r.Practice.PlatformRide != saved.Practice.PlatformRide || r.Arena().Platforms[0].X != saved.Arena().Platforms[0].X {
			t.Fatal("recovered ferry diverged")
		}
	}
	r.SetPaused(true, time.UnixMilli(r.LastMS))
	x, deck, clock := r.Player.X, r.Arena().Platforms[0].X, r.Clock
	r.Step(Input{X: 1}, time.UnixMilli(r.LastMS+200))
	if r.Player.X != x || r.Arena().Platforms[0].X != deck || r.Clock != clock {
		t.Fatal("paused ferry moved")
	}
	if err = r.ResetPractice(time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	if r.Practice.PlatformRide != 0 || r.Arena().Platforms[0].X != p.X || r.Clock != 0 {
		t.Fatal("reset retained ferry progress")
	}
}

func TestMovingPlatformCompletionRequiresRideAndCannotRewardCampaign(t *testing.T) {
	r := movingPlatformRun(t)
	r.Player.X = r.Practice.GoalX
	r.practiceTick()
	if r.Status != "fighting" {
		t.Fatal("walking around ferry completed drill")
	}
	r.Practice.PlatformRide = 250
	r.practiceTick()
	if r.Status != "complete" || !r.Practice.Completed {
		t.Fatal("ride and exit did not complete")
	}
	r.FinishCheckpoint("advance", nil)
	if r.NextRoom() || r.Gold != 0 || len(r.History) != 0 || len(r.CompletedLevels) != 0 || len(r.Drops) != 0 {
		t.Fatal("practice escaped isolation")
	}
}

func TestMovingPlatformCarrySweepsWallsAndCreditUsesActualTravel(t *testing.T) {
	r := movingPlatformRun(t)
	p := r.Arena().Platforms[0]
	r.Player.X, r.Player.Y = p.X+p.W/2, p.Y+p.H/2
	start := r.Player.X
	r.Practice.Arena.HighCover = []Obstacle{{X: start + 35, Y: 315, W: 20, H: 175}}
	r.Clock = 1
	r.practiceTick()
	if r.Player.X >= start+35 || r.Practice.PlatformRide != r.Player.X-start {
		t.Fatal("ferry carried through wall or credited blocked travel")
	}
}

func TestMovingPlatformReversesContinuouslyAndDoesNotCarryBystanders(t *testing.T) {
	r := movingPlatformRun(t)
	r.Clock = 7.99
	r.practiceTick()
	p := r.Arena().Platforms[0]
	r.Player.X, r.Player.Y = p.X+p.W/2, p.Y+p.H/2
	before := r.Player.X
	r.Clock = 8
	r.practiceTick()
	peak := r.Player.X
	r.Clock = 8.01
	r.practiceTick()
	if math.Abs(peak-before-.75) > 1e-7 || math.Abs(r.Player.X-before) > 1e-7 {
		t.Fatal("turnaround jumped or lost rider")
	}
	r.Player.Y = 330
	before = r.Player.X
	r.Clock = 8.02
	r.practiceTick()
	if r.Player.X != before || r.Player.Elevation != 0 {
		t.Fatal("nearby bystander moved with deck")
	}
}

func TestMovingPlatformRealStepsEarnRideThenExit(t *testing.T) {
	r := movingPlatformRun(t)
	// Walk aboard from the normal entrance, then stand on the ferry.
	for i := 0; i < 100 && r.Player.Elevation < 16; i++ {
		r.Step(Input{X: 1}, time.UnixMilli(r.LastMS+50))
	}
	if r.Player.Elevation != 16 {
		t.Fatal("walking did not board the deck")
	}
	for i := 0; i < 100 && r.Practice.PlatformRide < 250; i++ {
		r.Step(Input{}, time.UnixMilli(r.LastMS+50))
	}
	if r.Practice.PlatformRide != 250 {
		t.Fatal("real combat steps did not earn ride credit")
	}
	for i := 0; i < 150 && r.Status == "fighting"; i++ {
		r.Step(Input{X: 1}, time.UnixMilli(r.LastMS+50))
	}
	if r.Status != "complete" || r.Player.X < 1250 {
		t.Fatal("riding and walking to exit did not finish")
	}
	if r.Gold != 0 || r.Stats.Attacks != 0 || r.Stats.Jumps != 0 {
		t.Fatal("ferry manufactured rewards or input")
	}
}
