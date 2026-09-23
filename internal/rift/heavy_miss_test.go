package rift

import "testing"

func TestHeavyMissCreatesLongerCounterattackOpening(t *testing.T) {
	for _, reason := range []string{"distance", "lane", "jump", "cover", "hit"} {
		t.Run(reason, func(t *testing.T) {
			r := testRun()
			r.Player.X, r.Player.Y = 500, 410
			r.Enemies = []Actor{{Kind: "knight", X: 560, Y: 410, HP: 100, Windup: .01}}
			switch reason {
			case "distance":
				r.Player.X = 400
			case "lane":
				r.Player.Y = 460
			case "jump":
				r.Player.Jump = .5
			case "cover":
				r.Level = &Level{Rooms: []Arena{{Obstacles: []Obstacle{{530, 380, 10, 60}}}}}
			}
			r.enemyTick(0, .02)
			e := &r.Enemies[0]
			if reason == "hit" {
				if e.Cooldown != 1.6 || e.PoseTime != .4 {
					t.Fatal("landed strike recovery changed")
				}
				return
			}
			if e.Cooldown < 2 || e.PoseTime < .8 {
				t.Fatalf("miss has no extended recovery: %+v", e)
			}
			x, y := e.X, e.Y
			r.Player.X, r.Player.Y = 300, 450
			r.enemyTick(0, .3)
			if e.X != x || e.Y != y {
				t.Fatal("heavy enemy chased during missed swing recovery")
			}
			r.enemyTick(0, .6)
			if e.X == x && e.Y == y {
				t.Fatal("heavy enemy never resumed pursuit")
			}
		})
	}
}
