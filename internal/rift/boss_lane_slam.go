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
