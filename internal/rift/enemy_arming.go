package rift

import "math"

func blastContains(e, p *Actor) bool {
	dx, dy := (p.X-e.X)/115, (p.Y-e.Y)/55
	return dx*dx+dy*dy <= 1
}

func (r *Run) cancelEnemyArming(e *Actor) {
	if e.ArmingTimer <= 0 {
		return
	}
	e.ArmingTimer, e.Windup = 0, 0
	e.AttackName = ""
	e.Cooldown = math.Max(e.Cooldown, 1.4)
	r.eventAtHeight("arming_cancel", e.X, e.Y-30, 0, e.Elevation)
}

func (r *Run) tickEnemyArming(e *Actor, dt float64) bool {
	if !e.Explosive || e.Kind == "boss" || e.Kind == "treasure" || e.isObjectiveProp() {
		return false
	}
	if e.ArmingTimer > 0 {
		e.ArmingTimer = math.Max(0, math.Min(1.1, e.ArmingTimer)-dt)
		if e.ArmingTimer < 1e-9 {
			e.ArmingTimer = 0
		}
		e.Windup = e.ArmingTimer
		if e.ArmingTimer > 0 {
			e.Pose = "windup"
			return true
		}
		e.AttackName = ""
		e.Attacks++
		e.Pose = "attack"
		e.PoseTime = .35
		e.Cooldown = 2
		r.eventAtHeight("enemy_blast", e.X, e.Y, 0, e.Elevation)
		if blastContains(e, &r.Player) && r.clearMeleePath(e, &r.Player) {
			if r.Player.Jump >= .1 || r.SkillTimers["dodge_invulnerability"] > 0 {
				r.recordDodge()
			} else {
				r.hurtPlayerFromEnemy(math.Max(20, e.Damage), e.X, e.Y, e.ID)
			}
		}
		return true
	}
	dx, dy := (r.Player.X-e.X)/115, (r.Player.Y-e.Y)/55
	if e.Cooldown > 0 || e.Windup > 0 || e.PoseTime > 0 || e.Jump > 0 || dx*dx+dy*dy > .7 || !r.clearMeleePath(e, &r.Player) || !r.canStartEnemyAttack(e) {
		return false
	}
	e.ArmingTimer, e.Windup = 1.1, 1.1
	e.AttackName, e.Pose = "Blast", "windup"
	e.TargetX, e.TargetY = e.X, e.Y
	if e.Pack {
		r.PackAttackLockout = .45
		r.Stats.PackAttacks++
	}
	r.eventAtHeight("arming_start", e.X, e.Y-30, 0, e.Elevation)
	return true
}
