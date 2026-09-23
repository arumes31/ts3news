package rift

import "testing"

func TestProjectileTargetHitboxBoundaries(t *testing.T) {
	cases := []struct {
		name   string
		enemy  bool
		dx, dy float64
		hit    bool
	}{
		{"player horizontal inside", true, 24.9, 0, true}, {"player horizontal edge", true, 25, 0, false},
		{"player depth inside", true, 0, 22.9, true}, {"player depth edge", true, 0, 23, false},
		{"enemy horizontal inside", false, 34.9, 0, true}, {"enemy horizontal edge", false, 35, 0, false},
		{"enemy depth inside", false, 0, 29.9, true}, {"enemy depth edge", false, 0, 30, false},
		{"player excludes wider band", true, 30, 0, false}, {"enemy includes wider band", false, 30, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := testRun()
			r.Status = "cleared"
			r.Player.X = 500
			r.Player.Y = 410
			r.Enemies = []Actor{{ID: "target", X: 500, Y: 410, HP: 1000, MaxHP: 1000}}
			r.Projectiles = []Projectile{{X: 500 + tc.dx, Y: 410 + tc.dy, Enemy: tc.enemy, Life: 2, Power: 10, Skill: Skill{Kind: "fire"}}}
			hp := r.Player.HP
			r.tick(Input{}, 0)
			got := r.Enemies[0].HP < 1000
			if tc.enemy {
				got = r.Player.HP < hp
			}
			if got != tc.hit {
				t.Fatalf("hit=%v, want %v", got, tc.hit)
			}
		})
	}
}
