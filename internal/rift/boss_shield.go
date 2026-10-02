package rift

import "math"

// absorbBossShield consumes post-armor damage, returning only health overflow.
// Shield durability never contributes health-damage, kill or reward credit.
func (r *Run) absorbBossShield(e *Actor, damage float64) float64 {
	if damage <= 0 {
		return 0
	}
	absorbed := math.Min(e.BossShieldHP, damage)
	e.BossShieldHP -= absorbed
	r.eventAtHeight("boss_shield_hit", e.X, e.Y-30, absorbed, e.Elevation)
	if e.BossShieldHP == 0 {
		r.interruptEnemyCharge(e)
		e.ChargeRecovery = 0
		cancelEnemyMend(e)
		e.Windup = 0
		e.AttackName = ""
		e.WeakPoint = 1.2
		e.Pose = "stagger"
		e.PoseTime = 1.2
		e.Cooldown = math.Max(e.Cooldown, 2)
		r.eventAtHeight("boss_shield_break", e.X, e.Y-30, 0, e.Elevation)
	}
	return damage - absorbed
}
