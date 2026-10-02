package rift

import "math"

func bossSurgeTurn(e Actor) bool { return e.Kind == "boss" && e.Charging && e.Attacks%8 == 7 }
func (r *Run) releaseBossSurge(e *Actor) {
	r.event("boss_surge", e.X, e.Y, 0)
	if r.Player.Jump >= .1 || r.SkillTimers["dodge_invulnerability"] > 0 {
		r.recordDodge()
		e.WeakPoint = .8
		return
	}
	r.hurtPlayerFromEnemyGuardable(math.Max(32, e.Damage*1.4), e.X, e.Y, e.ID, false)
}
