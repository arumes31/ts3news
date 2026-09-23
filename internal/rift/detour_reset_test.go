package rift

import "testing"

func TestEnemyAbandonsDetourWhenPursuitIsClear(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		r := testRun()
		r.Player.X = 300
		r.Player.Y = 410
		r.Level = &Level{Rooms: []Arena{{}}}
		if blocked {
			r.Level.Rooms[0].Obstacles = []Obstacle{{400, 380, 30, 60}}
		}
		r.Enemies = []Actor{{ID: "pursuer", Kind: "goblin", X: 500, Y: 410, HP: 100, MaxHP: 100, RouteX: 385, RouteY: 454}}
		r.enemyTick(0, .05)
		if blocked {
			if r.Enemies[0].RouteY == 0 {
				t.Fatal("needed detour was discarded")
			}
		} else {
			if r.Enemies[0].RouteX != 0 || r.Enemies[0].RouteY != 0 || r.Enemies[0].Y != 410 {
				t.Fatal("obsolete detour redirected clear pursuit")
			}
		}
	}
}

func TestThirdStrikeClearsPreHitDetour(t *testing.T) {
	r := testRun()
	r.Status = "cleared"
	r.Player.X = 450
	r.Player.Y = 410
	r.Combo = 2
	r.Enemies = []Actor{{ID: "target", Kind: "goblin", X: 500, Y: 410, HP: 1000, MaxHP: 1000, RouteX: 700, RouteY: 454}}
	r.tick(Input{Attack: true}, 0)
	e := r.Enemies[0]
	if e.Knockdown <= 0 || e.X != 535 {
		t.Fatal("test did not apply knockback")
	}
	if e.RouteX != 0 || e.RouteY != 0 {
		t.Fatal("knockback retained a stale detour")
	}
}

func TestDetourRetainedWhenActorFootprintClipsCover(t *testing.T) {
	r := testRun()
	r.Player.X = 300
	r.Player.Y = 410
	r.Level = &Level{Rooms: []Arena{{Obstacles: []Obstacle{{400, 418, 30, 30}}}}}
	r.Enemies = []Actor{{ID: "pursuer", Kind: "goblin", X: 500, Y: 410, HP: 100, MaxHP: 100, RouteX: 385, RouteY: 400}}
	r.enemyTick(0, .05)
	if r.Enemies[0].RouteY == 0 {
		t.Fatal("center-only visibility discarded necessary body clearance")
	}
}
