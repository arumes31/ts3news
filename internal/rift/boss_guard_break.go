package rift

import "math"

// addBossStagger measures successful impacts independently of health and damage.
func (r *Run) addBossStagger(e *Actor, damage float64) {
	if e.Kind != "boss" {
		return
	}
	if e.HP <= 0 {
		e.BossStagger, e.BossGuardBreak, e.BossStaggerGrace = 0, 0, 0
		return
	}
	if damage <= 0 || e.BossGuardBreak > 0 || e.BossStaggerGrace > 0 {
		return
	}
	e.BossStagger = math.Min(100, math.Max(0, e.BossStagger)+20)
	if e.BossStagger < 100 {
		return
	}
	r.interruptEnemyCharge(e)
	e.ChargeRecovery = 0
	cancelEnemyMend(e)
	e.BossStagger = 0
	e.BossGuardBreak = 1.5
	e.BossStaggerGrace = 3.5
	e.Windup, e.AttackName = 0, ""
	e.Guard = false
	e.Pose, e.PoseTime = "stagger", 1.5
	e.Cooldown = math.Max(e.Cooldown, 2)
	r.eventAtHeight("boss_guard_break", e.X, e.Y-30, 0, e.Elevation)
}

func (r *Run) tickBossGuardBreak(e *Actor, dt float64) bool {
	if e.Kind != "boss" {
		return false
	}
	e.BossStaggerGrace = math.Max(0, math.Min(3.5, e.BossStaggerGrace)-dt)
	if e.BossGuardBreak <= 0 {
		return false
	}
	e.BossGuardBreak = math.Max(0, math.Min(1.5, e.BossGuardBreak)-dt)
	e.Windup, e.AttackName = 0, ""
	e.Guard = false
	e.Pose, e.PoseTime = "stagger", e.BossGuardBreak
	if e.BossGuardBreak == 0 {
		e.Pose = "idle"
	}
	return true
}
