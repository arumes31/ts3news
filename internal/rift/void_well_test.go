package rift

import (
	"math"
	"testing"
)

func TestVoidWellPullIsRadialBoundedAndDoesNotOvershoot(t *testing.T) {
	for _, tc := range []struct {
		name string
		zone Obstacle
		x, y float64
	}{
		{"ordinary", Obstacle{400, 340, 200, 100}, 450, 360},
		{"large saved zone", Obstacle{0, 315, 1500, 175}, 100, 350},
		{"near center", Obstacle{400, 340, 200, 100}, 499, 389},
		{"at center", Obstacle{400, 340, 200, 100}, 500, 390},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := circleTestRun()
			r.RoomObjective = nil
			r.Clock = 1.3
			r.Player.X = tc.x
			r.Player.Y = tc.y
			r.Level.Rooms[0] = Arena{Hazards: []Hazard{{Obstacle: tc.zone, Kind: "void", Period: 7, Duration: 1}}}
			dx, dy := tc.zone.X+tc.zone.W/2-tc.x, tc.zone.Y+tc.zone.H/2-tc.y
			distance := math.Hypot(dx, dy)
			scale := .3
			if distance*.3 > 24 {
				scale = 24 / distance
			}
			r.hazardTick()
			if math.Hypot(r.Player.X-tc.x, r.Player.Y-tc.y) > 24.000001 {
				t.Fatal("well exceeded pull cap")
			}
			if math.Abs(r.Player.X-(tc.x+dx*scale)) > 1e-8 || math.Abs(r.Player.Y-(tc.y+dy*scale)) > 1e-8 {
				t.Fatal("well did not pull radially toward center")
			}
		})
	}
}
func TestVoidWellPullRespectsSolidCover(t *testing.T) {
	r := circleTestRun()
	r.RoomObjective = nil
	r.Clock = 1.3
	r.Player.X = 480
	r.Player.Y = 380
	r.Level.Rooms[0] = Arena{HighCover: []Obstacle{{500, 340, 40, 100}}, Hazards: []Hazard{{Obstacle: Obstacle{400, 330, 300, 140}, Kind: "void", Period: 7, Duration: 1}}}
	r.hazardTick()
	if r.Player.X > 490 {
		t.Fatal("void pull crossed solid wall")
	}
}

func TestVoidWellCannotTunnelThinWallsOrForceLongLedgeDrops(t *testing.T) {
	for _, ledge := range []bool{false, true} {
		r := circleTestRun()
		r.RoomObjective = nil
		r.Clock = 1.3
		r.Player.X = 480
		r.Player.Y = 360
		arena := Arena{Hazards: []Hazard{{Obstacle: Obstacle{400, 330, 1100, 140}, Kind: "void", Period: 7, Duration: 1}}}
		if ledge {
			arena.Hazards[0].Obstacle = Obstacle{400, 330, 160, 140}
			arena.DropEdges = []DropEdge{{ID: "edge", X: 400, Y: 365, W: 300, LandingY: 425}}
		} else {
			arena.HighCover = []Obstacle{{491, 330, 1, 150}}
		}
		r.Level.Rooms[0] = arena
		r.hazardTick()
		if math.Hypot(r.Player.X-480, r.Player.Y-360) > 24.000001 {
			t.Fatal("terrain amplified pull beyond cap")
		}
		if !ledge && r.Player.X > 481 {
			t.Fatal("pull tunneled thin wall")
		}
	}
}
