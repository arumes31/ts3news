package rift

import "testing"

func TestEnemyProjectilesCannotSpawnInsideTallCover(t *testing.T) {
	for _, kind := range []string{"archer", "boss"} {
		for _, place := range []string{"inside", "boundary", "outside", "low-cover"} {
			t.Run(kind+"/"+place, func(t *testing.T) {
				r := testRun()
				r.Player.X, r.Player.Y = 200, 410
				wall := Obstacle{500, 380, 30, 60}
				arena := Arena{HighCover: []Obstacle{wall}}
				x := 515.0
				switch place {
				case "boundary":
					x = 500
				case "outside":
					x = 490
				case "low-cover":
					arena = Arena{Obstacles: []Obstacle{wall}}
				}
				r.Level = &Level{Rooms: []Arena{arena}}
				r.Enemies = []Actor{{Kind: kind, ArtKey: "catalog-enemy", X: x, Y: 410, HP: 100, Windup: .01, Attacks: 1, Shot: "fire"}}
				r.enemyTick(0, .02)
				blocked := place == "inside" || place == "boundary"
				if blocked {
					if len(r.Projectiles) != 0 {
						t.Fatal("projectile spawned inside tall cover")
					}
					for _, e := range r.Events {
						if e.Kind == "fire" {
							t.Fatal("cancelled shot emitted a firing effect")
						}
					}
					if r.Enemies[0].Windup != 0 || r.Enemies[0].Cooldown <= 0 {
						t.Fatal("cancelled shot must finish with a retry delay")
					}
				} else if len(r.Projectiles) != 1 {
					t.Fatal("clear origin or low cover incorrectly prevented firing")
				}
			})
		}
	}
}
