package rift

import (
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestThirdStrikeKnockbackRespectsTerrain(t *testing.T) {
	for _, width := range []float64{1, 40} {
		r := NewRunAtLevel("knockback", Build{HP: 200, Damage: 10}, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
		wall := Obstacle{240, 380, width, 60}
		r.Level.Rooms[0].Obstacles = []Obstacle{wall}
		r.Player.X = 160
		r.Player.Y = 410
		r.Player.Facing = 1
		r.Combo = 2
		r.Enemies = []Actor{{ID: "target", Kind: "goblin", X: 225, Y: 410, HP: 1000, MaxHP: 1000, Cooldown: 10}}
		r.tick(Input{Attack: true}, 0)
		e := r.Enemies[0]
		if e.Knockdown <= 0 {
			t.Fatal("third strike failed to knock down")
		}
		if contains(wall, e.X, e.Y, 10) || e.X > wall.X-10 {
			t.Fatalf("knockback crossed width-%v wall: x=%v", width, e.X)
		}
	}
}

func TestThirdStrikeKnockbackMovesInOpenGround(t *testing.T) {
	r := testRun()
	r.Status = "cleared"
	r.Level = nil
	r.Player.X = 100
	r.Player.Y = 410
	r.Combo = 2
	r.Enemies = []Actor{{ID: "target", Kind: "goblin", X: 150, Y: 410, HP: 1000, MaxHP: 1000}}
	r.tick(Input{Attack: true}, 0)
	if r.Enemies[0].X != 185 {
		t.Fatalf("open-ground knockback=%v, want 185", r.Enemies[0].X)
	}
}

func TestLeftwardThirdStrikeStopsBeforeWall(t *testing.T) {
	r := NewRunAtLevel("left-knockback", Build{HP: 200, Damage: 10}, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
	wall := Obstacle{239, 380, 1, 60}
	r.Level.Rooms[0].Obstacles = []Obstacle{wall}
	r.Player.X = 320
	r.Player.Y = 410
	r.Player.Facing = -1
	r.Combo = 2
	r.Enemies = []Actor{{ID: "target", Kind: "goblin", X: 255, Y: 410, HP: 1000, MaxHP: 1000, Cooldown: 10}}
	r.tick(Input{Attack: true}, 0)
	if r.Enemies[0].X < 250 || contains(wall, r.Enemies[0].X, r.Enemies[0].Y, 10) {
		t.Fatal("leftward knockback crossed terrain")
	}
}
