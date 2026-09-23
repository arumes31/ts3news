package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestTallCoverStopsProjectilesAtFirstImpact(t *testing.T) {
	for _, hostile := range []bool{false, true} {
		r := NewRunAtLevel("high-cover", Build{HP: 200}, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
		r.Status = "cleared"
		r.Player.X = 160
		r.Player.Y = 410
		var arena Arena
		if err := json.Unmarshal([]byte(`{"high_cover":[{"x":195,"y":380,"w":1,"h":60}]}`), &arena); err != nil {
			t.Fatal(err)
		}
		r.Level.Rooms[0] = arena
		r.Enemies = []Actor{{ID: "target", X: 230, Y: 410, HP: 1000, MaxHP: 1000}}
		shot := Projectile{X: 160, Y: 410, VX: 700, Life: 2, Power: 10, Skill: Skill{ID: "bolt", Kind: "fire"}}
		wantX := 195.0
		if hostile {
			shot.X = 230
			shot.VX = -700
			shot.Enemy = true
			wantX = 196
		}
		r.Projectiles = []Projectile{shot}
		r.tick(Input{}, .1)
		if r.Player.HP != 200 || r.Enemies[0].HP != 1000 || len(r.Projectiles) != 0 {
			t.Fatal("projectile crossed tall cover")
		}
		found := false
		for _, e := range r.Events {
			if e.Kind == "projectile_impact" {
				found = true
				if e.X != wantX {
					t.Fatalf("impact x=%v, want %v", e.X, wantX)
				}
			}
		}
		if !found {
			t.Fatal("missing cover impact effect")
		}
	}
}

func TestTallCoverBlocksJumpingAndPersists(t *testing.T) {
	r := NewRunAtLevel("pillar", Build{HP: 200}, time.Unix(100, 0), content.AbyssMobCatalog(), 3)
	if len(r.Level.Rooms[0].HighCover) == 0 {
		t.Fatal("pillar mission has no tall cover")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Arena().HighCover) == 0 {
		t.Fatal("save lost tall cover")
	}
	saved.Level.Rooms[0].Obstacles = nil
	saved.Level.Rooms[0].HighCover = []Obstacle{{195, 380, 10, 60}}
	for _, jump := range []float64{0, .5} {
		a := Actor{X: 180, Y: 410, Jump: jump}
		saved.moveActor(&a, 20, 0, false)
		if a.X != 180 {
			t.Fatalf("jump %v crossed tall cover", jump)
		}
	}
}

func TestProjectileUsesNearestTallCover(t *testing.T) {
	r := testRun()
	r.Status = "cleared"
	r.Enemies = nil
	r.Level = &Level{Rooms: []Arena{{HighCover: []Obstacle{{300, 380, 10, 60}, {200, 380, 10, 60}}}}}
	r.Projectiles = []Projectile{{X: 160, Y: 410, VX: 2000, Life: 2, Power: 10}}
	r.tick(Input{}, .1)
	for _, e := range r.Events {
		if e.Kind == "projectile_impact" {
			if e.X != 200 {
				t.Fatal("selected farther wall")
			}
			return
		}
	}
	t.Fatal("missing nearest-wall impact")
}
