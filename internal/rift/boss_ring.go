package rift

import "math"

// ringDanger is the same normalized ellipse and open quadrant drawn by the client.
// The inner edge and gap boundaries are safe; the outer edge belongs to the pulse.
func ringDanger(e Actor, x, y float64) bool {
	dx, dy := (x-e.TargetX)/200, (y-e.TargetY)/90
	q := dx*dx + dy*dy
	if q <= .25 || q > 1 {
		return false
	}
	if e.RingGap == 0 && dy <= -math.Abs(dx) || e.RingGap == 1 && dy >= math.Abs(dx) {
		return false
	}
	return true
}
func (r *Run) releaseBossRing(e *Actor) {
	// Target coordinates stay committed during the 2.3-second recovery, longer
	// than the impact reservation. The actor remains saved even after defeat.
	if r.SkillTimers == nil {
		r.SkillTimers = map[string]float64{}
	}
	r.SkillTimers["hazard-ring-"+e.ID] = .45
	r.event("boss_ring", e.TargetX, e.TargetY, float64(e.RingGap))
	if !ringDanger(*e, r.Player.X, r.Player.Y) {
		e.WeakPoint = .8
		return
	}
	if r.Player.Jump >= .1 || r.SkillTimers["dodge_invulnerability"] > 0 {
		r.recordDodge()
		e.WeakPoint = .8
		return
	}
	r.hurtPlayerFromEnemy(math.Max(32, e.Damage*1.4), e.TargetX, e.TargetY, e.ID)
}
