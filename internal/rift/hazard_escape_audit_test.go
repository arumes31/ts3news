package rift

import (
	"fmt"
	"testing"
)

func TestAllCampaignHazardsAllowEscapeWithActiveNeighbors(t *testing.T) {
	checked := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			for index, h := range arena.Hazards {
				if h.Disabled {
					continue
				}
				for _, fx := range []float64{.25, .5, .75} {
					for _, fy := range []float64{.25, .5, .75} {
						x, y := h.X+h.W*fx, h.Y+h.H*fy
						if x < 35 || x > 1565 || y < 315 || y > 490 {
							continue
						}
						blocked := false
						for _, wall := range arena.solidObstacles() {
							blocked = blocked || contains(wall, x, y, 10)
						}
						if blocked {
							continue
						}
						checked++
						t.Run(fmt.Sprintf("%d/%d/%d/%.0f,%.0f", level.ID, room, index, x, y), func(t *testing.T) {
							escaped := canWalkEscapeWithActiveNeighbors(arena, index, x, y, level.Region)
							if !escaped {
								t.Fatalf("no damage-free walking escape after 300ms reaction from %.1f,%.1f with all neighboring hazards active", x, y)
							}
						})
					}
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no hazard positions checked")
	}
	t.Logf("Checked %d legal hazard positions with active neighbors", checked)
}

func canWalkEscapeWithActiveNeighbors(arena Arena, index int, x, y float64, region int) bool {
	for _, direction := range []Input{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}, {X: 1, Y: 1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: -1, Y: -1}} {
		r := circleTestRun()
		r.RoomObjective = nil
		r.Level.Region = region
		copyArena := arena
		copyArena.Hazards = append([]Hazard(nil), arena.Hazards...)
		// Worst-case neighboring phases: active throughout the entire escape.
		// The focused hazard retains its full 1.2-second warning.
		for i := range copyArena.Hazards {
			copyArena.Hazards[i].Period = 10
			copyArena.Hazards[i].Duration = 8
			copyArena.Hazards[i].Offset = 1.21
		}
		copyArena.Hazards[index].Offset = 0
		r.Level.Rooms[0] = copyArena
		r.Player.X = x
		r.Player.Y = y
		r.Player.Elevation = arena.Elevation(x, y)
		r.Build.Armor = 0
		r.Clock = 0
		r.SkillTimers = map[string]float64{}
		before := r.Player.HP
		for frame := 0; frame < 61; frame++ {
			in := Input{}
			if frame >= 15 {
				in = direction
			}
			r.tick(in, .02)
		}
		safe := true
		for _, danger := range arena.Hazards {
			if !danger.Disabled && contains(danger.Obstacle, r.Player.X, r.Player.Y, 2) {
				safe = false
			}
		}
		if r.Player.HP == before && r.Stats.HazardContacts == 0 && safe && r.Stats.Jumps == 0 {
			return true
		}
	}
	return false
}

func TestEscapeAuditRejectsTrapsAndActiveNeighborContact(t *testing.T) {
	puddle := Hazard{Obstacle: Obstacle{500, 350, 100, 60}, Period: 5, Duration: 1, Kind: "fire"}
	for _, tc := range []struct {
		name  string
		arena Arena
		want  bool
	}{
		{"open exit", Arena{Hazards: []Hazard{puddle}}, true},
		{"whole floor", Arena{Hazards: []Hazard{{Obstacle: Obstacle{35, 315, 1530, 175}, Period: 5, Duration: 1, Kind: "fire"}}}, false},
		{"wall cage", Arena{Hazards: []Hazard{puddle}, HighCover: []Obstacle{{490, 340, 10, 80}, {600, 340, 10, 80}, {490, 340, 120, 10}, {490, 410, 120, 10}}}, false},
		{"active neighbor", Arena{Hazards: []Hazard{puddle, puddle}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := canWalkEscapeWithActiveNeighbors(tc.arena, 0, 550, 380, 0); got != tc.want {
				t.Fatalf("escape=%v want %v", got, tc.want)
			}
		})
	}
}
