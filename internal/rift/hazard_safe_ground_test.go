package rift

import (
	"math"
	"testing"
)

// A permanent route outside every enabled hazard is safe for every combination
// of phases, including combinations that offsets normally keep apart.
func permanentHazardSafeRoute(arena Arena) bool {
	r := &Run{Level: &Level{Rooms: []Arena{arena}}}
	type point struct{ x, y int }
	start, goal := point{160, 410}, point{1450, 320}
	safe := func(x, y float64) bool {
		for _, h := range arena.Hazards {
			if !h.Disabled && contains(h.Obstacle, x, y, 12) {
				return false
			}
		}
		for _, o := range arena.solidObstacles() {
			if contains(o, x, y, 12) {
				return false
			}
		}
		return x >= 35 && x <= 1565 && y >= 315 && y <= 490
	}
	if !safe(float64(start.x), float64(start.y)) || !safe(float64(goal.x), float64(goal.y)) {
		return false
	}
	queue := []point{start}
	seen := map[point]bool{start: true}
	for head := 0; head < len(queue); head++ {
		at := queue[head]
		if at == goal {
			return true
		}
		for _, step := range []point{{10, 0}, {-10, 0}, {0, 10}, {0, -10}} {
			next := point{at.x + step.x, at.y + step.y}
			if seen[next] || !safe(float64(next.x), float64(next.y)) {
				continue
			}
			a := Actor{ID: "player", X: float64(at.x), Y: float64(at.y)}
			clear := true
			// Check the path, not just the endpoints, and use actual collision/ledge rules.
			for part := 0; part < 5; part++ {
				x, y := a.X+float64(step.x)/5, a.Y+float64(step.y)/5
				if !safe(x, y) {
					clear = false
					break
				}
				r.moveActor(&a, float64(step.x)/5, float64(step.y)/5, false)
				if math.Abs(a.X-x) > .001 || math.Abs(a.Y-y) > .001 {
					clear = false
					break
				}
			}
			if clear {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	return false
}

func TestEveryCampaignRoomHasPermanentHazardSafeGround(t *testing.T) {
	rooms := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			rooms++
			if !permanentHazardSafeRoute(arena) {
				t.Fatalf("mission %d tier %d lacks a permanent safe entrance-to-exit route", level.ID, room+1)
			}
		}
	}
	if rooms != 300 {
		t.Fatalf("checked %d rooms, want 300", rooms)
	}
}

func TestSafeGroundValidatorRejectsCoverageAndDisconnectedRefuges(t *testing.T) {
	for _, tc := range []struct {
		name    string
		hazards []Hazard
		want    bool
	}{
		{"open ground", nil, true},
		{"isolated puddle", []Hazard{{Obstacle: Obstacle{500, 350, 100, 40}}}, true},
		{"whole floor", []Hazard{{Obstacle: Obstacle{35, 315, 1530, 175}}}, false},
		{"safe pockets separated by danger", []Hazard{{Obstacle: Obstacle{700, 315, 10, 175}}}, false},
		{"disabled barrier", []Hazard{{Obstacle: Obstacle{700, 315, 10, 175}, Disabled: true}}, true},
		{"thin strip between grid samples", []Hazard{{Obstacle: Obstacle{705, 315, 1, 175}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := permanentHazardSafeRoute(Arena{Hazards: tc.hazards}); got != tc.want {
				t.Fatalf("route=%v want %v", got, tc.want)
			}
		})
	}
}
