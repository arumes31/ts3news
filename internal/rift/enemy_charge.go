package rift

import "math"

const enemyChargeRecovery = .85

func (r *Run) finishEnemyCharge(e *Actor) {
	e.ChargeActive = false
	e.Windup = 0
	e.AttackName = ""
	e.ChargeRecovery = enemyChargeRecovery
	e.Cooldown = math.Max(e.Cooldown, 1.6)
	e.Pose = "recovery"
	e.PoseTime = 0
	e.RouteX, e.RouteY = 0, 0
	r.eventAtHeight("charge_recovery", e.X, e.Y-25, 0, e.Elevation)
}

func (r *Run) interruptEnemyCharge(e *Actor) {
	if e.ChargeActive || e.AttackName == "Charge" {
		pose, poseTime := e.Pose, e.PoseTime
		r.finishEnemyCharge(e)
		e.Pose, e.PoseTime = pose, poseTime
	}
}

// tickEnemyCharge consumes the enemy's action while aiming, rushing or recovering.
// The target is frozen at warning time; movement substeps cannot tunnel through cover.
func (r *Run) tickEnemyCharge(e *Actor, dt float64) bool {
	if !e.Charging || e.Kind == "boss" || e.Kind == "archer" || e.Kind == "treasure" {
		return false
	}
	if e.ChargeRecovery > 0 {
		e.ChargeRecovery = math.Max(0, math.Min(enemyChargeRecovery, e.ChargeRecovery)-dt)
		if e.PoseTime == 0 {
			e.Pose = "recovery"
		}
		if e.ChargeRecovery < 1e-9 {
			e.ChargeRecovery = 0
			if e.PoseTime == 0 {
				e.Pose = "idle"
			}
		}
		return true
	}
	p := &r.Player
	if e.AttackName == "Charge" && !e.ChargeActive {
		if e.Windup <= 0 {
			r.finishEnemyCharge(e)
			return true
		}
		e.Windup = math.Max(0, e.Windup-dt)
		if e.Windup <= 1e-9 {
			e.Windup = 0
			e.ChargeActive = true
			e.Attacks++
			e.Pose = "attack"
			e.PoseTime = 1
			r.eventAtHeight("charge_rush", e.X, e.Y-25, 0, e.Elevation)
		}
		return true
	}
	if e.ChargeActive {
		remaining := 360 * dt
		for remaining > 0 {
			dx, dy := e.TargetX-e.X, e.TargetY-e.Y
			distance := math.Hypot(dx, dy)
			if distance < 1 {
				r.finishEnemyCharge(e)
				return true
			}
			step := math.Min(6, math.Min(distance, remaining))
			sx, sy := dx/distance*step, dy/distance*step
			beforeX, beforeY := e.X, e.Y
			r.moveActor(e, sx, sy, false)
			if math.Abs(e.X-beforeX-sx) > .01 || math.Abs(e.Y-beforeY-sy) > .01 {
				r.finishEnemyCharge(e)
				return true
			}
			remaining -= step
			if math.Abs(p.X-e.X) < 30 && math.Abs(p.Y-e.Y) < 20 && r.clearMeleePath(e, p) {
				if p.Jump >= .25 || r.SkillTimers["dodge_invulnerability"] > 0 {
					r.recordDodge()
				} else {
					r.hurtPlayerFromEnemy(math.Max(19, e.Damage), e.X, e.Y, e.ID)
				}
				r.finishEnemyCharge(e)
				return true
			}
		}
		return true
	}
	dx, dy := p.X-e.X, p.Y-e.Y
	distance := math.Hypot(dx, dy)
	minRange, maxRange := 110.0, 320.0
	if e.Elite && e.Enraged {
		minRange, maxRange = 85, 360
	}
	if e.Cooldown > 0 || e.Windup > 0 || e.PoseTime > 0 || distance < minRange || distance > maxRange || math.Abs(dy) > 100 || !r.canStartEnemyAttack(e) {
		return false
	}
	e.TargetX, e.TargetY = p.X, p.Y
	e.RouteX, e.RouteY = 0, 0
	e.Facing = math.Copysign(1, dx)
	e.AttackName = "Charge"
	e.Windup = .7
	e.Pose = "windup"
	if e.Pack {
		r.PackAttackLockout = .45
		r.Stats.PackAttacks++
	}
	r.eventAtHeight("charge_warning", e.X, e.Y-25, 0, e.Elevation)
	return true
}
