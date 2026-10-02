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

func channelDanger(e Actor, x, y float64) bool {
	dx, dy := (x-e.TargetX)/125, (y-e.TargetY)/62
	return dx*dx+dy*dy <= 1
}

func (r *Run) releaseBossChannel(e *Actor) {
	if r.SkillTimers == nil {
		r.SkillTimers = map[string]float64{}
	}
	r.SkillTimers["hazard-channel-"+e.ID] = .45
	r.event("boss_channel_pulse", e.TargetX, e.TargetY, 0)
	if !channelDanger(*e, r.Player.X, r.Player.Y) {
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
