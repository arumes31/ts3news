package rift

import (
	"encoding/json"
	"math"
	"testing"
)

func shieldTurnRun() *Run {
	r := testRun()
	r.Player.X, r.Player.Y = 460, 410
	r.Enemies = []Actor{{ID: "shield", Name: "Shield guard", Kind: "knight", Shield: true, X: 500, Y: 410, Facing: 1, HP: 100, MaxHP: 100, Cooldown: 2}}
	return r
}

func TestShieldGuardTurningPersistsAndAllowsRearStrike(t *testing.T) {
	r := shieldTurnRun()
	r.enemyTick(0, .1)
	e := &r.Enemies[0]
	if e.Facing != 1 || !e.Guard || e.Pose != "guard" || e.TurnDelay <= 0 || e.Windup != 0 {
		t.Fatalf("no guard flank window: %+v", e)
	}
	r.enemyTick(0, .1)
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	saved.enemyTick(0, .1)
	if saved.Enemies[0].Facing != 1 {
		t.Fatal("save/reload shortened turn window")
	}
	saved.enemyTick(0, .05)
	if saved.Enemies[0].Facing != -1 || saved.Enemies[0].TurnDelay != 0 {
		t.Fatal("shield never completed its turn")
	}
	r.hurtEnemyPiercing(0, 20, "hit", 0)
	if r.Stats.RearStrikes != 1 || math.Abs(r.Enemies[0].HP-81) > 1e-9 {
		t.Fatalf("rear opening did not bypass armor: %+v", r.Enemies[0])
	}
}

func TestShieldGuardTurnResetsWhenPlayerReturnsInFront(t *testing.T) {
	r := shieldTurnRun()
	r.enemyTick(0, .2)
	r.Player.X = 540
	r.enemyTick(0, .05)
	if r.Enemies[0].Facing != 1 || r.Enemies[0].TurnDelay != 0 {
		t.Fatal("front return did not cancel turn")
	}
	r.Player.X = 460
	r.enemyTick(0, .2)
	if r.Enemies[0].Facing != 1 {
		t.Fatal("old turn time leaked into new flank")
	}
	r.enemyTick(0, .15)
	if r.Enemies[0].Facing != -1 {
		t.Fatal("fresh turn did not finish")
	}
}

func TestShieldTurningOnlySlowsGuardingRecovery(t *testing.T) {
	for _, mode := range []string{"ordinary", "windup", "far", "hit", "ready"} {
		t.Run(mode, func(t *testing.T) {
			r := shieldTurnRun()
			e := &r.Enemies[0]
			switch mode {
			case "ordinary":
				e.Kind = "goblin"
				e.Shield = false
			case "windup":
				e.Windup = .5
			case "far":
				r.Player.X = 250
			case "hit":
				e.Pose = "hit"
				e.PoseTime = .3
			case "ready":
				e.Cooldown = 0
			}
			r.enemyTick(0, .05)
			if e.Facing != -1 || e.Guard || e.TurnDelay != 0 {
				t.Fatalf("non-guard turn slowed: %+v", e)
			}
		})
	}
}

func TestShieldCarrierLegacyAndFlaggedRolesBoundTurnDelay(t *testing.T) {
	for _, kind := range []string{"knight", "goblin"} {
		t.Run(kind, func(t *testing.T) {
			r := shieldTurnRun()
			e := &r.Enemies[0]
			e.Kind = kind
			e.Shield = kind == "goblin"
			e.TurnDelay = 1000
			r.enemyTick(0, .1)
			if !e.Guard || e.Facing != 1 || math.Abs(e.TurnDelay-.25) > 1e-9 {
				t.Fatalf("unbounded or missing shield window: %+v", e)
			}
			r.Player.Y = 480
			r.enemyTick(0, .05)
			if e.Guard || e.TurnDelay != 0 || e.Facing != -1 {
				t.Fatal("leaving guard range retained turn delay")
			}
		})
	}
}

func TestShieldGuardInterruptionCancelsPendingTurn(t *testing.T) {
	for _, pose := range []string{"stagger", "knockdown"} {
		t.Run(pose, func(t *testing.T) {
			r := shieldTurnRun()
			r.enemyTick(0, .1)
			e := &r.Enemies[0]
			e.Pose = pose
			e.PoseTime = .4
			if pose == "knockdown" {
				e.Knockdown = .4
			}
			r.enemyTick(0, .05)
			if e.Guard || e.TurnDelay != 0 {
				t.Fatalf("interrupted guard retained pending turn: %+v", e)
			}
		})
	}
}
