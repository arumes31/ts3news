package rift

import "math"

const bossLaneHeight = 175.0 / 3

func bossLane(y float64) int {
	for lane := 2; lane > 0; lane-- {
		if y >= 315+float64(lane)*bossLaneHeight {
			return lane
		}
	}
	return 0
}

func (r *Run) releaseLaneSlam(e *Actor) {
	if e.SlamLane < 0 || e.SlamLane > 2 {
		return
	}
	if r.SkillTimers == nil {
		r.SkillTimers = map[string]float64{}
	}
	r.SkillTimers[bossLaneReservationKeys[e.SlamLane]] = .4
	r.event("lane_slam", 800, 315+(float64(e.SlamLane)+.5)*bossLaneHeight, float64(e.SlamLane))
	if bossLane(r.Player.Y) != e.SlamLane {
		e.WeakPoint = .8
		return
	}
	if r.SkillTimers["dodge_invulnerability"] > 0 || r.Player.Jump >= .1 {
		r.recordDodge()
		e.WeakPoint = .8
		return
	}
	r.hurtPlayerFromEnemy(math.Max(32, e.Damage*1.4), e.X, e.Y, e.ID)
}

var bossLaneReservationKeys = [3]string{"hazard-slam-lane-0", "hazard-slam-lane-1", "hazard-slam-lane-2"}

// reservedBossLane excludes every simultaneous danger band. Reservations only
// suppress arena hazard contact; other actors and projectiles remain dangerous.
func (r *Run) reservedBossLane(y float64) bool {
	lane := bossLane(y)
	reserved := false
	for danger, key := range bossLaneReservationKeys {
		if r.SkillTimers[key] > 0 {
			if lane == danger {
				return false
			}
			reserved = true
		}
	}
	for _, e := range r.Enemies {
		if e.HP <= 0 || e.Kind != "boss" || !e.LaneSlams || e.AttackName != "Lane Slam" || e.Windup <= 0 || e.SlamLane < 0 || e.SlamLane > 2 {
			continue
		}
		if lane == e.SlamLane {
			return false
		}
		reserved = true
	}
	return reserved
}
