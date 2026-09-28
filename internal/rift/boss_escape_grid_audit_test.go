//go:build brawl_audit

package rift

import (
	"math"
	"testing"
)

var laneAuditDirections = [][2]float64{{0, -1}, {0, 1}, {-1, -1}, {1, -1}, {-1, 1}, {1, 1}, {-1, 0}, {1, 0}}

func auditEscapeRoute(r *Run, start Actor, speed float64, steps, turn int, first, second [2]float64, safe func(Actor) bool) bool {
	r.Player = start
	for step := 0; step < steps; step++ {
		direction := first
		if step >= turn {
			direction = second
		}
		scale := speed * .02 / math.Hypot(direction[0], direction[1])
		r.moveActor(&r.Player, direction[0]*scale, direction[1]*scale*.6, false)
		if safe(r.Player) {
			return true
		}
	}
	return false
}

func auditLaneEscape(r *Run, start Actor, speed float64, steps int) (straight, escaped bool) {
	lane := bossLane(start.Y)
	return auditEscape(r, start, speed, steps, func(a Actor) bool { return bossLane(a.Y) != lane })
}

func auditEscape(r *Run, start Actor, speed float64, steps int, safe func(Actor) bool) (straight, escaped bool) {
	for _, direction := range laneAuditDirections {
		if auditEscapeRoute(r, start, speed, steps, steps, direction, direction, safe) {
			return true, true
		}
	}
	for turn := 1; turn < steps; turn++ {
		for _, first := range laneAuditDirections {
			for _, second := range laneAuditDirections {
				if auditEscapeRoute(r, start, speed, steps, turn, first, second, safe) {
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
						legal := arena.groundPath(x, y, x, y, 10)
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

// Exercise every authored panel combination with the wave gate open and closed.
// These are geometric lane-escape probes, not spawned boss encounters.
func TestCampaignWaveFloorLaneEscapeGridAudit(t *testing.T) {
	warning := (&Run{}).NextBossAttack(Actor{Kind: "boss", LaneSlams: true}).Windup
	steps := int(math.Round((warning - .3) / .02))
	for _, speed := range []float64{235, 141} {
		checked, blocked, needTurn, unresolved, states := 0, 0, 0, 0, 0
		for _, level := range Campaign() {
			for room, frozen := range level.Rooms {
				if len(frozen.FragileFloor) == 0 {
					continue
				}
				for mask := 0; mask < 1<<len(frozen.FragileFloor); mask++ {
					for _, closed := range []bool{false, true} {
						r := Run{Level: &level, Room: room, RoomObjective: &RoomObjective{Kind: "survive_waves"}}
						for i, box := range frozen.FragileFloor {
							r.RoomObjective.FloorSegments = append(r.RoomObjective.FloorSegments, WaveFloorSegment{Obstacle: box, Collapsed: mask&(1<<i) != 0})
						}
						if frozen.WaveGate != nil {
							r.RoomObjective.Gate = &WaveGate{Obstacle: *frozen.WaveGate, Closed: closed}
						}
						arena := r.Arena()
						states++
						for x := 35.0; x <= 1565; x += 30 {
							for _, y := range []float64{315, 335, 355, 375, 395, 415, 435, 455, 475, 490} {
								legal := arena.groundPath(x, y, x, y, 10)
								for _, wall := range arena.solidObstacles() {
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
										t.Logf("NO_TESTED_ESCAPE speed=%.0f mission=%d room=%d panels=%b gate=%v x=%.0f y=%.0f", speed, level.ID, room, mask, closed, x, y)
									}
								}
							}
						}
					}
				}
			}
		}
		if states != 80 {
			t.Fatalf("expected 80 authored floor/gate states, got %d", states)
		}
		if unresolved > 0 {
			t.Errorf("%d floor-state starts have no tested escape at speed %.0f", unresolved, speed)
		}
		t.Logf("FLOOR_GRID_RESULT speed=%.0f states=%d checked=%d blocked=%d no_straight_escape=%d no_tested_escape=%d", speed, states, checked, blocked, needTurn, unresolved)
	}
}

// Require the protected center/gap, not merely the unprotected exterior.
func TestCampaignRingShelterGridAudit(t *testing.T) {
	warning := (&Run{}).NextBossAttack(Actor{Kind: "boss", RingAttack: true, Attacks: 2}).Windup
	steps := int(math.Round((warning - .3) / .02))
	for _, speed := range []float64{235, 141} {
		checked, centers, needTurn, unresolved := 0, 0, 0, 0
		for _, level := range Campaign() {
			r := Run{Level: &level, Room: 2}
			arena := r.Arena()
			legal := func(x, y, radius float64) bool {
				if x < 35 || x > 1565 || y < 315 || y > 490 || !arena.groundPath(x, y, x, y, radius) {
					return false
				}
				for _, wall := range arena.solidObstacles() {
					if contains(wall, x, y, radius) {
						return false
					}
				}
				return true
			}
			for _, x := range []float64{200, 500, 800, 1100, 1400} {
				for _, y := range []float64{350, 402.5, 455} {
					boss := Actor{Kind: "boss", HP: 100, RingAttack: true, Windup: warning, AttackName: "Void Ring", TargetX: x, TargetY: y}
					if y < 402.5 {
						boss.RingGap = 1
					}
					if !legal(x, y, actorClearance(&boss)) {
						continue
					}
					centers++
					r.Enemies = []Actor{boss}
					for _, dx := range []float64{-190, -150, -110, -70, 0, 70, 110, 150, 190} {
						for _, dy := range []float64{-80, -40, 0, 40, 80} {
							start := Actor{ID: "player", X: x + dx, Y: y + dy}
							if !legal(start.X, start.Y, 10) || !ringDanger(boss, start.X, start.Y) {
								continue
							}
							checked++
							straight, escaped := auditEscape(&r, start, speed, steps, func(a Actor) bool { return r.reservedBossArea(a.X, a.Y) })
							if !straight {
								needTurn++
							}
							if !escaped {
								unresolved++
								if unresolved <= 20 {
									t.Logf("NO_TESTED_SHELTER speed=%.0f mission=%d center=%.1f,%.1f start=%.1f,%.1f", speed, level.ID, x, y, start.X, start.Y)
								}
							}
						}
					}
				}
			}
		}
		if checked == 0 {
			t.Fatal("no ring danger samples")
		}
		if unresolved > 0 {
			t.Errorf("%d ring starts have no tested protected escape at speed %.0f", unresolved, speed)
		}
		t.Logf("RING_GRID_RESULT speed=%.0f centers=%d checked=%d no_straight_escape=%d no_tested_shelter=%d", speed, centers, checked, needTurn, unresolved)
	}
}
