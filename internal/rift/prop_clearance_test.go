package rift

import (
	"fmt"
	"math"
	"testing"
)

// Walk a complete perimeter using production movement, with two units of spare
// clearance beyond the player's body. Hazards are traversable, not solid props.
func walkPropPerimeter(r *Run, o Obstacle, reverse bool) error {
	a := Actor{ID: "player"}
	margin := actorClearance(&a) + 2
	type point struct{ x, y float64 }
	corners := []point{{o.X - margin, o.Y - margin}, {o.X + o.W + margin, o.Y - margin}, {o.X + o.W + margin, o.Y + o.H + margin}, {o.X - margin, o.Y + o.H + margin}}
	if reverse {
		corners[1], corners[3] = corners[3], corners[1]
	}
	a.X, a.Y = corners[0].x, corners[0].y
	for _, target := range append(corners, corners[0]) {
		if target.x < 35 || target.x > Width-35 || target.y < 315 || target.y > 490 {
			return fmt.Errorf("perimeter leaves arena at %.0f/%.0f", target.x, target.y)
		}
		for math.Abs(a.X-target.x) > .001 || math.Abs(a.Y-target.y) > .001 {
			dx, dy := clamp(target.x-a.X, -2, 2), clamp(target.y-a.Y, -2, 2)
			wantX, wantY := a.X+dx, a.Y+dy
			for _, other := range r.Arena().solidObstacles() {
				if contains(other, wantX, wantY, margin) {
					return fmt.Errorf("spare clearance lost at %.0f/%.0f", wantX, wantY)
				}
			}
			r.moveActor(&a, dx, dy, false)
			if math.Abs(a.X-wantX) > .001 || math.Abs(a.Y-wantY) > .001 {
				return fmt.Errorf("walk blocked near %.0f/%.0f, got %.0f/%.0f", wantX, wantY, a.X, a.Y)
			}
		}
	}
	return nil
}

func TestAllCampaignPropsHaveWalkingClearanceOnEverySide(t *testing.T) {
	props := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			r := &Run{Level: &level, Room: room}
			for index, o := range arena.solidObstacles() {
				props++
				for _, reverse := range []bool{false, true} {
					if err := walkPropPerimeter(r, o, reverse); err != nil {
						t.Fatalf("mission %d room %d prop %d reverse=%v: %v", level.ID, room, index, reverse, err)
					}
				}
			}
		}
	}
	if props == 0 {
		t.Fatal("campaign props missing")
	}
	t.Logf("checked both directions around %d props in 300 rooms", props)
}

func TestPropClearanceCheckRejectsNarrowGapsAndArenaEdges(t *testing.T) {
	for _, tc := range []struct {
		name          string
		first, second Obstacle
		pass          bool
	}{
		{"comfortable gap", Obstacle{400, 380, 80, 40}, Obstacle{504, 380, 80, 40}, true},
		{"body only gap", Obstacle{400, 380, 80, 40}, Obstacle{500, 380, 80, 40}, false},
		{"overlap", Obstacle{400, 380, 80, 40}, Obstacle{470, 380, 80, 40}, false},
		{"boundary pinching", Obstacle{400, 320, 80, 40}, Obstacle{800, 380, 80, 40}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &Run{Level: &Level{Rooms: []Arena{{Obstacles: []Obstacle{tc.first, tc.second}}}}}
			for _, reverse := range []bool{false, true} {
				if err := walkPropPerimeter(r, tc.first, reverse); (err == nil) != tc.pass {
					t.Fatalf("reverse=%v pass=%v error=%v", reverse, tc.pass, err)
				}
			}
		})
	}
}
