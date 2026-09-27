package rift

import "math"

func (r *Run) interruptBossChannel(e *Actor) {
	if e.Kind != "boss" || e.AttackName != "Time Pulse" || e.Windup <= 0 {
		return
	}
	e.Windup = 0
	e.AttackName = ""
	e.Attacks++
	e.Cooldown = math.Max(e.Cooldown, 2.3)
	e.WeakPoint = 1
	r.event("boss_channel_interrupt", e.X, e.Y, 0)
}

func (r *Run) releaseBossChannel(e *Actor) {
	r.event("boss_channel_pulse", e.TargetX, e.TargetY, 0)
	dx, dy := (r.Player.X-e.TargetX)/125, (r.Player.Y-e.TargetY)/62
	if dx*dx+dy*dy > 1 {
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
