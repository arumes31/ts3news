package rift

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestEnemyRoutesRemainDeterministicAcrossMidDetourSaves(t *testing.T) {
	for _, kind := range []string{"goblin", "knight", "archer", "boss", "treasure"} {
		t.Run(kind, func(t *testing.T) {
			r := testRun()
			r.Level = &Level{Rooms: []Arena{{Obstacles: []Obstacle{{400, 380, 40, 60}}, HighCover: []Obstacle{{650, 350, 30, 55}}}}}
			r.Player.X, r.Player.Y = 100, 410
			r.Enemies = []Actor{{ID: "pursuer", Kind: kind, X: 500, Y: 410, HP: 1000, MaxHP: 1000, Speed: 100}}
			clone := func(source *Run) *Run {
				data, err := json.Marshal(source)
				if err != nil {
					t.Fatal(err)
				}
				var saved Run
				if err := json.Unmarshal(data, &saved); err != nil {
					t.Fatal(err)
				}
				return &saved
			}
			saved := clone(r)
			detoured, moved := false, false
			for step := 0; step < 240; step++ {
				// Save during ongoing movement rather than restarting the encounter.
				if step%17 == 0 {
					saved = clone(saved)
				}
				x, y := 100.0, 410.0
				if step >= 80 {
					x, y = 800, 430
				}
				if step >= 160 {
					x, y = 450, 340
				}
				for _, run := range []*Run{r, saved} {
					run.Player.X, run.Player.Y = x, y
					if step == 110 {
						run.knockbackActor(&run.Enemies[0], -25, 8)
					}
					run.enemyTick(0, .025)
				}
				detoured = detoured || r.Enemies[0].RouteY != 0
				moved = moved || r.Enemies[0].X != 500 || r.Enemies[0].Y != 410
				if !reflect.DeepEqual(r.Enemies, saved.Enemies) {
					t.Fatalf("route diverged after save at step %d: live=%+v restored=%+v", step, r.Enemies[0], saved.Enemies[0])
				}
			}
			if !moved {
				t.Fatal("fixture never exercised movement")
			}
			if kind != "treasure" && !detoured {
				t.Fatal("fixture never exercised obstacle detours")
			}
		})
	}
}
