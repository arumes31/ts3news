package rift

import "testing"

func TestNavigationClearanceMatchesCreatureSize(t *testing.T) {
	for _, tc := range []struct {
		kind string
		gap  float64
		pass bool
	}{
		{"treasure", 18, true}, {"wolf", 18, true}, {"goblin", 18, false},
		{"goblin", 28, true}, {"boss", 28, false}, {"boss", 44, true},
	} {
		r := testRun()
		r.Level = &Level{Rooms: []Arena{{Obstacles: []Obstacle{{400, 350, 40, 50}, {400, 400 + tc.gap, 40, 60}}}}}
		a := Actor{Kind: tc.kind, X: 350, Y: 400 + tc.gap/2}
		target := Actor{X: 480, Y: a.Y}
		if r.clearPursuitPath(&a, &target) != tc.pass {
			t.Fatalf("%s pursuit clearance incorrect for gap %v", tc.kind, tc.gap)
		}
		for i := 0; i < 60; i++ {
			r.moveActor(&a, 2, 0, false)
		}
		if (a.X > 440) != tc.pass {
			t.Fatalf("%s movement clearance incorrect for gap %v: x=%v", tc.kind, tc.gap, a.X)
		}
	}
}

func TestLargeCreatureDetourKeepsBodyClearance(t *testing.T) {
	r := testRun()
	wall := Obstacle{400, 380, 40, 60}
	r.Level = &Level{Rooms: []Arena{{Obstacles: []Obstacle{wall}}}}
	a := Actor{Kind: "boss", X: 370, Y: 410}
	r.moveActor(&a, 12, 0, true)
	if a.RouteY < 458 || a.RouteX < 458 {
		t.Fatalf("large creature detour hugs obstacle: %+v", a)
	}
	for i := 0; i < 25; i++ {
		r.moveActor(&a, 4, 0, true)
		if contains(wall, a.X, a.Y, 18) {
			t.Fatal("large creature clipped cover during detour")
		}
	}
	if a.X <= 440 {
		t.Fatal("large creature failed to finish detour")
	}
}
