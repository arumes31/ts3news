//go:build brawl_audit

package rift

import (
	"math"
	"testing"
)

// Time Pulse locks onto the player's warning-start location. Probe legal starts
// against authored terrain and all authored wave-floor/gate combinations.
// This is finite geometry coverage, not a simulation of concurrent attackers.
func TestCampaignChannelShelterGridAudit(t *testing.T) {
	warning := (&Run{}).NextBossAttack(Actor{Kind: "boss", LaneSlams: true, Attacks: 3}).Windup
	steps := int(math.Round((warning - .3) / .02))
	for _, speed := range []float64{235, 141} {
		checked, blocked, states, needTurn, unresolved := 0, 0, 0, 0, 0
		for _, level := range Campaign() {
			for room, frozen := range level.Rooms {
				variants := 1
				if len(frozen.FragileFloor) > 0 {
					variants = 2 * (1 << len(frozen.FragileFloor))
				}
				for state := 0; state < variants; state++ {
					r := Run{Level: &level, Room: room}
					if variants > 1 {
						r.RoomObjective = &RoomObjective{Kind: "survive_waves"}
						for i, box := range frozen.FragileFloor {
							r.RoomObjective.FloorSegments = append(r.RoomObjective.FloorSegments, WaveFloorSegment{Obstacle: box, Collapsed: (state/2)&(1<<i) != 0})
						}
						if frozen.WaveGate != nil {
							r.RoomObjective.Gate = &WaveGate{Obstacle: *frozen.WaveGate, Closed: state%2 != 0}
						}
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
							r.Enemies = []Actor{{ID: "channel", Kind: "boss", HP: 100, LaneSlams: true, AttackName: "Time Pulse", Windup: warning, TargetX: x, TargetY: y}}
							straight, escaped := auditEscape(&r, Actor{ID: "player", X: x, Y: y}, speed, steps, func(a Actor) bool { return r.reservedBossArea(a.X, a.Y) })
							if !straight {
								needTurn++
							}
							if !escaped {
								unresolved++
								if unresolved <= 20 {
									t.Logf("NO_TESTED_SHELTER speed=%.0f mission=%d room=%d state=%d start=%.0f,%.0f", speed, level.ID, room, state, x, y)
								}
							}
						}
					}
				}
			}
		}
		if states != 370 {
			t.Fatalf("expected370 terrain states, got%d", states)
		}
		if unresolved > 0 {
			t.Errorf("%d sampled starts lack a protected channel escape at speed%.0f", unresolved, speed)
		}
		t.Logf("CHANNEL_GRID_RESULT speed=%.0f states=%d checked=%d blocked=%d no_straight_escape=%d no_tested_shelter=%d", speed, states, checked, blocked, needTurn, unresolved)
	}
}
