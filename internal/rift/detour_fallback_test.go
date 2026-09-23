package rift

import "testing"

func TestEnemyUsesOtherSideWhenPreferredDetourExitsArena(t *testing.T) {
	for _, top := range []bool{false, true} {
		wall := Obstacle{400, 410, 40, 80}
		y := 460.0
		if top {
			wall = Obstacle{400, 315, 40, 80}
			y = 340
		}
		r := testRun()
		r.Level = &Level{Rooms: []Arena{{Obstacles: []Obstacle{wall}}}}
		a := Actor{Kind: "goblin", X: 385, Y: y}
		for i := 0; i < 60; i++ {
			r.moveActor(&a, 4, 0, true)
		}
		if a.X <= 440 {
			t.Fatalf("enemy stalled at blocked preferred passage: %+v", a)
		}
		if contains(wall, a.X, a.Y, 10) {
			t.Fatal("fallback crossed solid terrain")
		}
	}
}

func TestEnemySwitchesDetourWhenPassageCloses(t *testing.T) {
	r := testRun()
	wall := Obstacle{400, 380, 40, 60}
	r.Level = &Level{Rooms: []Arena{{Obstacles: []Obstacle{wall}}}}
	a := Actor{Kind: "goblin", X: 385, Y: 410}
	r.moveActor(&a, 4, 0, true)
	if a.RouteY <= 440 {
		t.Fatal("fixture did not choose lower passage")
	}
	r.Level.Rooms[0].HighCover = []Obstacle{{380, 445, 80, 45}}
	r.moveActor(&a, 4, 0, true)
	if a.RouteY >= 380 {
		t.Fatalf("enemy retained closed lower passage: %+v", a)
	}
	for i := 0; i < 60; i++ {
		r.moveActor(&a, 4, 0, true)
		for _, o := range r.Arena().solidObstacles() {
			if contains(o, a.X, a.Y, 10) {
				t.Fatal("fallback crossed terrain")
			}
		}
	}
	if a.X <= 460 {
		t.Fatal("enemy never completed alternative route")
	}
}
