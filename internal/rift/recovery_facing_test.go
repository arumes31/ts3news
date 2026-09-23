package rift

import "testing"

func TestHeavyRecoveryTurnsWithoutMovingOrAttacking(t *testing.T) {
	for _, side := range []float64{-1, 1} {
		r := testRun()
		r.Player.X, r.Player.Y = 500+side*100, 410
		r.Enemies = []Actor{{Kind: "knight", X: 500, Y: 410, HP: 100, Facing: -side, Pose: "attack", PoseTime: .8, Cooldown: 2.2, Attacks: 1}}
		r.enemyTick(0, .1)
		e := r.Enemies[0]
		if e.Facing != side {
			t.Fatal("recovering enemy did not turn toward player")
		}
		if e.X != 500 || e.Y != 410 || e.Attacks != 1 || e.Windup != 0 || e.Pose != "attack" || e.PoseTime < .69 {
			t.Fatal("turning interrupted stationary recovery")
		}
		r.Player.X = 500 - side*100
		r.enemyTick(0, .1)
		if r.Enemies[0].Facing != -side {
			t.Fatal("recovery facing did not follow player crossing")
		}
	}
}

func TestRecoveryFacingDoesNotOverrideDisablingStates(t *testing.T) {
	for _, pose := range []string{"stagger", "knockdown"} {
		r := testRun()
		r.Player.X, r.Player.Y = 400, 410
		r.Enemies = []Actor{{Kind: "knight", X: 500, Y: 410, HP: 100, Facing: 1, Pose: pose, PoseTime: .8}}
		if pose == "knockdown" {
			r.Enemies[0].Knockdown = .8
		}
		r.enemyTick(0, .1)
		if r.Enemies[0].Facing != 1 {
			t.Fatal("disabled enemy turned during control effect")
		}
	}
}
