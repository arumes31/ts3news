package rift

import "testing"

func TestBossPhaseChangesWindupWithoutRemovingCounterplay(t *testing.T) {
	for _, tc := range []struct {
		phase  int
		windup float64
	}{{1, 1.15}, {2, .98}, {3, .85}} {
		r := testRun()
		r.Player.X, r.Player.Y = 500, 410
		r.Enemies = []Actor{{ID: "boss", Kind: "boss", ArtKey: "monster:test", X: 560, Y: 410, HP: 1000, MaxHP: 1000, Phase: tc.phase}}
		r.enemyTick(0, .01)
		if r.Enemies[0].Windup != tc.windup {
			t.Fatalf("phase %d windup=%v, want %v", tc.phase, r.Enemies[0].Windup, tc.windup)
		}
		hp := r.Player.HP
		r.enemyTick(0, tc.windup-.05)
		if r.Enemies[0].Attacks != 0 || r.Player.HP != hp {
			t.Fatal("boss attacked before phase warning ended")
		}
		r.Player.Jump = .5
		r.enemyTick(0, .06)
		if r.Enemies[0].Attacks != 1 || r.Player.HP != hp || r.Enemies[0].Cooldown != 2.3 {
			t.Fatal("phase timing removed avoidance or recovery")
		}
	}
}
