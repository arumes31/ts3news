package rift

import (
	"encoding/json"
	"math"
	"testing"
)

func TestShortcutBarrierOpensSavedDirectRoute(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			for _, c := range arena.Cover {
				if !c.Shortcut {
					continue
				}
				count++
				r := testRun()
				r.Level = &level
				r.Room = room
				r.Player.X, r.Player.Y = c.X-80, 410
				start := r.Player.X
				for step := 0; step < 50; step++ {
					r.moveActor(&r.Player, 4, 0, false)
				}
				if r.Player.X >= c.X {
					t.Fatal("intact shortcut crossed")
				}
				r.Player.X, r.Player.Y, r.Player.Facing = c.X-25, 410, 1
				if !r.attackTerrainCover(30) {
					t.Fatal("barrier cannot be attacked from shortcut")
				}
				if !r.attackTerrainCover(30) {
					t.Fatal("barrier cannot be destroyed by second attack from shortcut")
				}
				raw, err := json.Marshal(r)
				if err != nil {
					t.Fatal(err)
				}
				var saved Run
				if err = json.Unmarshal(raw, &saved); err != nil {
					t.Fatal(err)
				}
				saved.Player.X, saved.Player.Y = start, 410
				for step := 0; step < 50; step++ {
					saved.moveActor(&saved.Player, 4, 0, false)
				}
				if math.Abs(saved.Player.X-start-200) > .001 {
					t.Fatal("broken shortcut remains blocked")
				}
			}
		}
	}
	if count != 10 {
		t.Fatalf("shortcut count %d", count)
	}
}
func TestShortcutIntactBypassesRemainWalkable(t *testing.T) {
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			for _, c := range arena.Cover {
				if !c.Shortcut {
					continue
				}
				for _, y := range []float64{320, 480} {
					r := testRun()
					r.Level = &level
					r.Room = room
					r.Player.X, r.Player.Y = c.X-80, 410
					targets := [][2]float64{{c.X - 80, y}, {c.X + c.W + 40, y}, {c.X + c.W + 40, 410}}
					for _, target := range targets {
						for i := 0; i < 150; i++ {
							dx, dy := target[0]-r.Player.X, target[1]-r.Player.Y
							if math.Hypot(dx, dy) < .01 {
								break
							}
							r.moveActor(&r.Player, clamp(dx, -3, 3), clamp(dy, -3, 3), false)
						}
						if math.Hypot(r.Player.X-target[0], r.Player.Y-target[1]) > .01 {
							t.Fatalf("mission%d bypass blocked", level.ID)
						}
					}
				}
			}
		}
	}
}
