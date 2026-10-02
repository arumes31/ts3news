package rift

import (
	"fmt"
	"math"
	"testing"
	"time"

	"ts3news/internal/content"
)

func TestDiagonalPillarCoverProvidesShotShelterAndBypasses(t *testing.T) {
	for id := 3; id <= LevelCount; id += 10 {
		r := NewRunAtLevel("diagonal-cover", Build{HP: 1000}, time.Unix(0, 0), content.AbyssMobCatalog(), id)
		for room := 0; room < 3; room++ {
			t.Run(fmt.Sprintf("%d/%d", id, room), func(t *testing.T) {
				r.Room = room
				r.RoomObjective = nil
				walls := r.Arena().HighCover
				if len(walls) != 3 {
					t.Fatalf("want three diagonal shot blockers, got %d", len(walls))
				}
				direction := math.Copysign(1, walls[1].Y-walls[0].Y)
				for i, wall := range walls {
					if i > 0 && (wall.X <= walls[i-1].X || direction*(wall.Y-walls[i-1].Y) < 20) {
						t.Fatal("pillars do not form a diagonal")
					}
					left, right := Actor{X: wall.X - 40, Y: wall.Y + wall.H/2}, Actor{X: wall.X + wall.W + 40, Y: wall.Y + wall.H/2}
					if r.clearProjectilePath(&left, &right) {
						t.Fatal("pillar does not block a direct shot")
					}
					left.Y, right.Y = wall.Y-24, wall.Y-24
					if !r.clearProjectilePath(&left, &right) {
						t.Fatal("no firing angle around pillar")
					}
				}
				for _, kind := range []string{"player", "boss"} {
					for _, y := range []float64{315, 490} {
						actor := Actor{ID: kind, Kind: kind, X: 35, Y: y}
						for step := 0; step < 383; step++ {
							r.moveActor(&actor, 4, 0, false)
						}
						if actor.X < 1564 || actor.Y != y {
							t.Fatalf("%s bypass blocked at %v,%v", kind, actor.X, actor.Y)
						}
					}
				}
			})
		}
	}
}

func TestArchersCanFireFromEveryDiagonalArenaSpawn(t *testing.T) {
	for id := 3; id <= LevelCount; id += 10 {
		r := NewRunAtLevel("diagonal-archer", Build{HP: 1e9}, time.Unix(0, 0), content.AbyssMobCatalog(), id)
		for room, spawns := range r.EncounterPlan {
			for index, spawn := range spawns {
				t.Run(fmt.Sprintf("%d/%d/%d", id, room, index), func(t *testing.T) {
					r.Room = room
					r.RoomObjective = nil
					r.Projectiles = nil
					// Probe archer navigation at each authored location without inheriting
					// an unrelated patrol or pack role from the original monster.
					archer := Actor{ID: spawn.ID, Name: "Archer probe", Kind: "archer", X: spawn.X, Y: spawn.Y, HP: 100, MaxHP: 100, Speed: 90, Facing: -1, Pose: "idle", Shot: "arrow"}
					r.Enemies = []Actor{archer}
					for step := 0; step < 1800 && len(r.Projectiles) == 0; step++ {
						r.enemyTick(0, .02)
					}
					if len(r.Projectiles) == 0 {
						t.Fatalf("archer cannot acquire a firing lane from %v,%v; ended %v,%v", spawn.X, spawn.Y, r.Enemies[0].X, r.Enemies[0].Y)
					}
					if !r.clearProjectilePath(&r.Enemies[0], &r.Player) {
						t.Fatal("archer fired through solid cover")
					}
				})
			}
		}
	}
}
