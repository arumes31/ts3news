package rift

import (
	"math"
	"testing"
)

func TestArcherRetreatsFromNearbyPlayer(t *testing.T) {
	for _, side := range []float64{-1, 1} {
		r := testRun()
		r.Player.X, r.Player.Y = 500, 410
		r.Enemies = []Actor{{Kind: "archer", X: 500 + side*70, Y: 410, HP: 100, Speed: 80}}
		r.enemyTick(0, .05)
		e := r.Enemies[0]
		if math.Abs(e.X-r.Player.X) <= 70 || e.Windup != 0 || e.Pose != "run" || e.Facing != -side {
			t.Fatalf("archer did not backpedal facing player: %+v", e)
		}
	}
}

func TestArcherRetreatPreservesCommittedShot(t *testing.T) {
	r := testRun()
	r.Player.X, r.Player.Y = 500, 410
	r.Enemies = []Actor{{Kind: "archer", X: 570, Y: 410, HP: 100, Windup: .01}}
	r.enemyTick(0, .02)
	if len(r.Projectiles) != 1 || r.Enemies[0].X != 570 {
		t.Fatal("close player cancelled or moved a committed shot")
	}
}

func TestArcherPinnedAtBoundaryStillAttacks(t *testing.T) {
	for _, x := range []float64{35, Width - 35} {
		r := testRun()
		r.Player.X, r.Player.Y = x+70, 410
		if x > Width/2 {
			r.Player.X = x - 70
		}
		r.Enemies = []Actor{{Kind: "archer", X: x, Y: 410, HP: 100}}
		r.enemyTick(0, .05)
		if r.Enemies[0].Windup == 0 {
			t.Fatal("pinned archer stalled instead of attacking")
		}
	}
}

func TestArcherRetreatStopsAtFiringDistance(t *testing.T) {
	r := testRun()
	r.Player.X, r.Player.Y = 500, 410
	r.Enemies = []Actor{{Kind: "archer", X: 570, Y: 410, HP: 100, Speed: 80}}
	for i := 0; i < 30 && r.Enemies[0].Windup == 0; i++ {
		r.enemyTick(0, .05)
	}
	e := r.Enemies[0]
	if e.Windup == 0 || e.X < 650 || e.X > 654 {
		t.Fatalf("archer did not stop retreating to aim: %+v", e)
	}
}

func TestArcherRetreatCannotPassThroughCover(t *testing.T) {
	r := testRun()
	r.Player.X, r.Player.Y = 500, 410
	r.Level = &Level{Rooms: []Arena{{Obstacles: []Obstacle{{590, 380, 10, 60}}}}}
	r.Enemies = []Actor{{Kind: "archer", X: 579, Y: 410, HP: 100, Speed: 80}}
	r.enemyTick(0, .05)
	if r.Enemies[0].X >= 590 || r.Enemies[0].Windup == 0 {
		t.Fatal("blocked retreat must stay outside cover and allow an attack")
	}
}
