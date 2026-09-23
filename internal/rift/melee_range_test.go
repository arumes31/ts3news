package rift

import (
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestBasicMeleeDirectionalBoundaries(t *testing.T) {
	cases := []struct {
		name           string
		forward, depth float64
		hit            bool
	}{
		{"rear inside", -9.9, 0, true}, {"rear boundary", -10, 0, true}, {"rear outside", -10.1, 0, false},
		{"front inside", 94.9, 0, true}, {"front boundary", 95, 0, false}, {"front outside", 95.1, 0, false},
		{"depth inside", 20, 31.9, true}, {"depth boundary", 20, 32, false}, {"depth outside", 20, 32.1, false},
		{"negative depth inside", 20, -31.9, true}, {"negative depth boundary", 20, -32, false},
	}
	for _, facing := range []float64{-1, 1} {
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				r := NewRunAtLevel("range", Build{HP: 200, Damage: 10}, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
				r.Status = "cleared"
				r.Level.Rooms[0].Obstacles = nil
				r.Player.X = 500
				r.Player.Y = 410
				r.Player.Facing = facing
				r.Enemies = []Actor{{ID: "target", HP: 1000, MaxHP: 1000, X: 500 + tc.forward*facing, Y: 410 + tc.depth}}
				r.tick(Input{Attack: true}, 0)
				if got := r.Enemies[0].HP < 1000; got != tc.hit {
					t.Fatalf("facing %v, forward %v, depth %v: hit=%v, want %v", facing, tc.forward, tc.depth, got, tc.hit)
				}
			})
		}
	}
}
