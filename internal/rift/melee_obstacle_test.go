package rift

import (
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestSolidObstacleBlocksMeleeBothWays(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		r := NewRunAtLevel("melee-wall", Build{HP: 200, Damage: 10}, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
		r.Player.X = 160
		r.Player.Y = 410
		r.Player.Facing = 1
		r.Level.Rooms[0].Obstacles = nil
		if blocked {
			r.Level.Rooms[0].Obstacles = []Obstacle{{195, 380, 10, 60}}
		}
		r.Enemies = []Actor{{ID: "target", Kind: "goblin", X: 230, Y: 410, HP: 1000, MaxHP: 1000, Cooldown: 10}}
		r.tick(Input{Attack: true}, 0)
		if got := r.Enemies[0].HP < 1000; got == blocked {
			t.Fatalf("player melee blocked=%v, hit=%v", blocked, got)
		}
		r.Enemies[0].Pose = "windup"
		r.Enemies[0].PoseTime = 0
		r.Enemies[0].Windup = .01
		hp := r.Player.HP
		r.enemyTick(0, .02)
		if got := r.Player.HP < hp; got == blocked {
			t.Fatalf("enemy melee blocked=%v, hit=%v", blocked, got)
		}
	}
}

func TestMeleeObstacleSegmentGeometry(t *testing.T) {
	r := NewRunAtLevel("melee-geometry", Build{HP: 200}, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
	r.Level.Rooms[0].Obstacles = []Obstacle{{200, 400, 20, 20}}
	cases := []struct {
		name           string
		x1, y1, x2, y2 float64
		clear          bool
	}{
		{"horizontal", 180, 410, 240, 410, false}, {"vertical", 210, 380, 210, 440, false},
		{"diagonal", 180, 380, 240, 440, false}, {"above", 180, 390, 240, 390, true},
		{"diagonal miss", 180, 390, 230, 395, true}, {"edge", 180, 400, 240, 400, false},
		{"beyond target", 150, 410, 190, 410, true}, {"same point outside", 150, 410, 150, 410, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, b := Actor{X: tc.x1, Y: tc.y1}, Actor{X: tc.x2, Y: tc.y2}
			if r.clearMeleePath(&a, &b) != tc.clear || r.clearMeleePath(&b, &a) != tc.clear {
				t.Fatal("incorrect segment visibility")
			}
		})
	}
}
