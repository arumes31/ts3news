package rift

import (
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestLowCoverAllowsFriendlyAndHostileProjectiles(t *testing.T) {
	for _, hostile := range []bool{false, true} {
		r := NewRunAtLevel("low-cover", Build{HP: 200, Damage: 10}, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
		r.Status = "cleared"
		r.Player.X = 160
		r.Player.Y = 410
		r.Level.Rooms[0].Obstacles = []Obstacle{{195, 380, 10, 60}}
		r.Enemies = []Actor{{ID: "target", X: 230, Y: 410, HP: 1000, MaxHP: 1000}}
		shot := Projectile{X: 160, Y: 410, VX: 700, Life: 2, Power: 10, Skill: Skill{ID: "bolt", Kind: "fire"}}
		if hostile {
			shot.X = 230
			shot.VX = -700
			shot.Enemy = true
		}
		r.Projectiles = []Projectile{shot}
		r.tick(Input{}, .1)
		if hostile && r.Player.HP >= 200 || !hostile && r.Enemies[0].HP >= 1000 {
			t.Fatalf("low cover incorrectly stopped hostile=%v projectile", hostile)
		}
		if len(r.Projectiles) != 0 {
			t.Fatal("projectile failed to resolve its target hit")
		}
	}
}
