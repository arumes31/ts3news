package rift

import "testing"

func TestEnemyRecalculatesPursuitAfterPlayerJumpsAcrossCover(t *testing.T) {
	for _, direction := range []float64{-1, 1} {
		r := testRun()
		r.Level = &Level{Rooms: []Arena{{Obstacles: []Obstacle{{400, 380, 30, 60}}}}}
		r.Player.X, r.Player.Y = 360, 410
		enemyX := 460.0
		if direction < 0 {
			r.Player.X, enemyX = 470, 370
		}
		r.Enemies = []Actor{{Kind: "goblin", X: enemyX, Y: 410, HP: 100, MaxHP: 100}}
		r.enemyTick(0, .2)
		if r.Enemies[0].RouteY == 0 {
			t.Fatal("fixture did not establish a detour around cover")
		}
		r.Enemies[0].Speed = 20
		r.tick(Input{Jump: true}, .02)
		for i := 0; i < 24; i++ {
			r.tick(Input{X: direction}, .02)
		}
		if r.Player.Jump <= .1 || (direction > 0 && r.Player.X <= 440) || (direction < 0 && r.Player.X >= 390) {
			t.Fatalf("player did not jump across cover: x=%v jump=%v", r.Player.X, r.Player.Jump)
		}
		if r.Enemies[0].RouteX != 0 || r.Enemies[0].RouteY != 0 {
			t.Fatalf("pursuer retained obsolete detour: %+v", r.Enemies[0])
		}
		for i := 0; i < 15; i++ {
			r.tick(Input{}, .02)
		}
		if r.Player.Jump != 0 {
			t.Fatal("player did not land")
		}
		if r.Enemies[0].RouteY != 0 {
			t.Fatal("landing reinstated obsolete detour")
		}
	}
}
