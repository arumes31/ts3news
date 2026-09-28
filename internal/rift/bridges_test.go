package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
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

func TestBridgeSafePlacementAndPursuit(t *testing.T) {
	r := dropEdgeTestRun()
	r.Level.Rooms[0] = Arena{Bridges: []NarrowBridge{{Obstacle: Obstacle{400, 370, 200, 90}, ID: "bridge"}}}
	for _, kind := range []string{"goblin", "wolf", "boss"} {
		actor := Actor{Kind: kind, X: 500, Y: 340}
		if !r.Arena().settleEnemySpawn(&actor) {
			t.Fatal("safe spawn not found")
		}
		if !r.Arena().groundPath(actor.X, actor.Y, actor.X, actor.Y, actorClearance(&actor)) {
			t.Fatalf("%s spawned beside deck", kind)
		}
	}
	if r.safeDropLanding(&r.Player, 500, 340) {
		t.Fatal("ledge landing allowed in bridge gap")
	}
	if !r.safeDropLanding(&r.Player, 500, 410) {
		t.Fatal("safe deck landing rejected")
	}
	from, to := Actor{X: 300, Y: 340}, Actor{X: 700, Y: 340}
	if r.clearPursuitPath(&from, &to) {
		t.Fatal("burrow or pursuit can cross bridge gap")
	}
	if !r.clearProjectilePath(&from, &to) {
		t.Fatal("empty bridge gap blocks projectiles")
	}
	from.Y, to.Y = 410, 410
	if !r.clearPursuitPath(&from, &to) {
		t.Fatal("deck pursuit blocked")
	}
}

func TestBridgePursuersReachOppositeBank(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		r := dropEdgeTestRun()
		r.Level.Rooms[0] = Arena{Bridges: []NarrowBridge{{Obstacle: Obstacle{400, 370, 200, 90}, ID: "bridge"}}}
		actor := Actor{Kind: "goblin", X: 300, Y: 340}
		target := Actor{X: 700, Y: 470}
		if reverse {
			actor.X, actor.Y, target.X, target.Y = 700, 470, 300, 340
		}
		for i := 0; i < 200; i++ {
			dx, dy := clamp(target.X-actor.X, -5, 5), clamp(target.Y-actor.Y, -5, 5)
			r.moveActor(&actor, dx, dy, true)
			if !r.Arena().groundPath(actor.X, actor.Y, actor.X, actor.Y, actorClearance(&actor)) {
				t.Fatal("pursuer left deck")
			}
		}
		if actor.X != target.X || actor.Y != target.Y {
			t.Fatalf("pursuer stalled: %+v", actor)
		}
	}
}

func TestCampaignBridgesHaveClearApproachesAndIsolatedSaves(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		arena := level.Rooms[0]
		if len(arena.Bridges) == 0 {
			continue
		}
		count++
		if !arena.ValidBridges() {
			t.Fatalf("invalid mission %d bridge", level.ID)
		}
		bridge := arena.Bridges[0]
		if sw := arena.HazardSwitch; sw != nil && !arena.groundPath(sw.X, sw.Y, sw.X, sw.Y, 10) { t.Fatal("switch placed in bridge gap") }
		run := NewRunAtLevel("bridge-campaign", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level.ID)
		if !arena.groundPath(run.Player.X, run.Player.Y, run.Player.X, run.Player.Y, 10) {
			t.Fatal("entrance in gap")
		}
		for _, enemy := range run.Enemies {
			if !arena.groundPath(enemy.X, enemy.Y, enemy.X, enemy.Y, actorClearance(&enemy)) {
				t.Fatal("enemy spawned in gap")
			}
		}
		for _, wall := range arena.solidObstacles() {
			if wall.X < bridge.X+bridge.W+40 && wall.X+wall.W > bridge.X-40 {
				t.Fatal("bridge approach obstructed")
			}
		}
		for _, h := range arena.Hazards {
			if h.X < bridge.X+bridge.W+40 && h.X+h.W > bridge.X-40 {
				t.Fatal("bridge approach has hazard")
			}
		}
		run.Player.X, run.Player.Y = bridge.X+bridge.W/2, bridge.Y+bridge.H/2
		if run.FloorMaterial() != "wood" {
			t.Fatal("bridge footsteps use wrong surface")
		}
		copyLevel := cloneCampaignLevel(level)
		copyLevel.Rooms[0].Bridges[0].Y++
		if level.Rooms[0].Bridges[0].Y != bridge.Y {
			t.Fatal("saved bridge aliases catalog")
		}
	}
	if count != 10 {
		t.Fatalf("authored %d bridge rooms, want 10", count)
	}
}
