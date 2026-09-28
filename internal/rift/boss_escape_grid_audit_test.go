//go:build brawl_audit

package rift

import (
	"math"
	"testing"
)

var laneAuditDirections = [][2]float64{{0, -1}, {0, 1}, {-1, -1}, {1, -1}, {-1, 1}, {1, 1}, {-1, 0}, {1, 0}}

func auditLaneRoute(r *Run, start Actor, speed float64, steps, turn int, first, second [2]float64) bool {
	r.Player = start
	lane := bossLane(start.Y)
	for step := 0; step < steps; step++ {
		direction := first
		if step >= turn {
			direction = second
		}
		scale := speed * .02 / math.Hypot(direction[0], direction[1])
		r.moveActor(&r.Player, direction[0]*scale, direction[1]*scale*.6, false)
		if bossLane(r.Player.Y) != lane {
			return true
		}
	}
	return false
}

func auditLaneEscape(r *Run, start Actor, speed float64, steps int) (straight, escaped bool) {
	for _, direction := range laneAuditDirections {
		if auditLaneRoute(r, start, speed, steps, steps, direction, direction) {
			return true, true
		}
	}
	for turn := 1; turn < steps; turn++ {
		for _, first := range laneAuditDirections {
			for _, second := range laneAuditDirections {
				if auditLaneRoute(r, start, speed, steps, turn, first, second) {
					return false, true
				}
			}
		}
	}
	return false, false
}

// This finite diagnostic probes geometry, not combat or hazard timing. It tests
// eight held directions, followed by every single-turn combination if needed.
// It is not proof for unsampled coordinates or more complicated routes.
func TestCampaignLaneEscapeGridAudit(t *testing.T) {
	warning := (&Run{}).NextBossAttack(Actor{Kind: "boss", LaneSlams: true}).Windup
	steps := int(math.Round((warning - .3) / .02))
	for _, speed := range []float64{235, 141} {
		checked, blocked, needTurn, unresolved := 0, 0, 0, 0
		for _, level := range Campaign() {
			for room, arena := range level.Rooms {
				r := Run{Level: &Level{Rooms: []Arena{arena}}}
				obstacles := arena.solidObstacles()
				for x := 35.0; x <= 1565; x += 30 {
					for _, y := range []float64{315, 335, 355, 375, 395, 415, 435, 455, 475, 490} {
						legal := true
						for _, wall := range obstacles {
							if contains(wall, x, y, 10) {
								legal = false
								break
							}
						}
						if !legal {
							blocked++
							continue
						}
						checked++
						straight, escaped := auditLaneEscape(&r, Actor{ID: "player", X: x, Y: y}, speed, steps)
						if !straight {
							needTurn++
						}
						if !escaped {
							unresolved++
							if unresolved <= 20 {
								t.Logf("NO_TESTED_ESCAPE speed=%.0f mission=%d room=%d x=%.0f y=%.0f lane=%d", speed, level.ID, room, x, y, bossLane(y))
							}
						}
					}
				}
			}
		}
		if unresolved > 0 {
			t.Errorf("%d sampled positions have no tested escape at speed %.0f within %.2fs", unresolved, speed, warning-.3)
		}
		t.Logf("GRID_RESULT speed=%.0f checked=%d blocked=%d no_straight_escape=%d no_tested_escape=%d", speed, checked, blocked, needTurn, unresolved)
	}
}
