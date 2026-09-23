package rift

import (
	"testing"
	"time"
)

func TestProjectileImpactEvent(t *testing.T) {
	for _, mode := range []string{"friendly", "hostile", "expired", "miss"} {
		t.Run(mode, func(t *testing.T) {
			r := testRun()
			r.Enemies = []Actor{{ID: "target", X: r.Player.X + 100, Y: r.Player.Y, HP: 1000, MaxHP: 1000, Cooldown: 10}}
			shot := Projectile{X: r.Enemies[0].X, Y: r.Player.Y, Power: 10, Life: 2, Skill: r.Build.Skills[0]}
			if mode == "hostile" {
				shot.Enemy = true
				shot.X = r.Player.X
			}
			if mode == "expired" {
				shot.Life = 0
			}
			if mode == "miss" {
				shot.Y += 100
			}
			r.Projectiles = []Projectile{shot}
			r.Step(Input{}, time.Unix(100, 100_000_000))
			count := 0
			for _, event := range r.Events {
				if event.Kind == "projectile_impact" {
					count++
					if event.X != shot.X || event.Y != shot.Y {
						t.Fatal("impact moved away from collision")
					}
				}
			}
			want := 0
			if mode == "friendly" || mode == "hostile" {
				want = 1
			}
			if count != want {
				t.Fatalf("impact events=%d, want %d", count, want)
			}
			if want == 1 && len(r.Projectiles) != 0 {
				t.Fatal("colliding projectile remained live")
			}
		})
	}
}
