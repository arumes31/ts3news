package rift

import (
	"encoding/json"
	"testing"
)

func TestBridgeSweptGroundBounds(t *testing.T) {
	arena := Arena{Bridges: []NarrowBridge{{Obstacle: Obstacle{400, 370, 200, 90}, ID: "bridge"}}}
	for _, tc := range []struct {
		name                   string
		x1, y1, x2, y2, radius float64
		want                   bool
	}{
		{"deck", 300, 410, 700, 410, 10, true},
		{"reverse deck", 700, 410, 300, 410, 10, true},
		{"upper gap leap", 300, 340, 700, 340, 10, false},
		{"reverse gap leap", 700, 340, 300, 340, 10, false},
		{"lower gap", 500, 410, 500, 470, 10, false},
		{"upper boundary", 500, 410, 500, 380, 10, true},
		{"upper footprint", 500, 410, 500, 379, 10, false},
		{"outside span", 300, 330, 300, 480, 10, true},
		{"diagonal gap", 300, 410, 700, 480, 10, false},
		{"boss footprint", 500, 410, 500, 385, 18, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := arena.groundPath(tc.x1, tc.y1, tc.x2, tc.y2, tc.radius); got != tc.want {
				t.Fatalf("ground path=%v want %v", got, tc.want)
			}
		})
	}
	raw, err := json.Marshal(arena)
	if err != nil {
		t.Fatal(err)
	}
	var recovered Arena
	if err = json.Unmarshal(raw, &recovered); err != nil {
		t.Fatal(err)
	}
	if recovered.Bridges[0] != arena.Bridges[0] || recovered.groundPath(300, 340, 700, 340, 10) {
		t.Fatal("saved bridge bounds lost")
	}
	if !(Arena{}).groundPath(300, 340, 700, 340, 10) {
		t.Fatal("legacy arena changed")
	}
}

func TestBridgeMovementAndKnockbackRespectDeckWhileAirborne(t *testing.T) {
	for _, jump := range []float64{0, 1} {
		r := dropEdgeTestRun()
		r.Level.Rooms[0] = Arena{Bridges: []NarrowBridge{{Obstacle: Obstacle{400, 370, 200, 90}, ID: "bridge"}}}
		r.Player.X, r.Player.Y, r.Player.Jump = 500, 410, jump
		r.moveActor(&r.Player, 0, -100, false)
		if r.Player.Y != 410 {
			t.Fatal("movement left deck")
		}
		r.knockbackActor(&r.Player, 0, 200)
		if r.Player.Y > 450 {
			t.Fatal("knockback left deck")
		}
		r.Player.X, r.Player.Y = 300, 340
		r.moveActor(&r.Player, 400, 0, false)
		if r.Player.X != 300 {
			t.Fatal("large move skipped bridge gap")
		}
		r.Player.Y = 410
		r.moveActor(&r.Player, 400, 0, false)
		if r.Player.X != 700 {
			t.Fatal("deck crossing blocked")
		}
	}
}
