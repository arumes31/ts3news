package rift

import "math"

func cancelEnemyMend(e *Actor) {
	if e.HealTarget == "" {
		return
	}
	e.HealTarget = ""
	e.Windup = 0
	e.AttackName = ""
	if e.Pose == "cast" && e.PoseTime == 0 {
		e.Pose = "idle"
	}
}

func (r *Run) validMendTarget(healer, target *Actor) bool {
	return healer != target && target.ID != "" && target.HP > 0 && target.MaxHP > 0 && target.HP < target.MaxHP && target.Kind != "treasure" && !target.isObjectiveProp() && math.Hypot(healer.X-target.X, healer.Y-target.Y) <= 320 && r.clearProjectilePath(healer, target)
}

func (r *Run) tickEnemyMend(i int, dt float64) bool {
	e := &r.Enemies[i]
	if !e.Healer || e.Kind == "boss" || e.Kind == "treasure" {
		return false
	}
	if e.HealTarget != "" {
		target := -1
		for j := range r.Enemies {
			if r.Enemies[j].ID == e.HealTarget && r.validMendTarget(e, &r.Enemies[j]) {
				target = j
				break
			}
		}
		if target < 0 {
			cancelEnemyMend(e)
			return true
		}
		if e.Windup <= 0 {
			cancelEnemyMend(e)
			return true
		}
		e.Windup = math.Max(0, e.Windup-dt)
		if e.Windup > 1e-9 {
			return true
		}
		ally := &r.Enemies[target]
		amount := math.Min(ally.MaxHP-ally.HP, math.Min(30, ally.MaxHP*.12))
		ally.HP += amount
		r.eventAtHeight("heal", ally.X, ally.Y-30, amount, ally.Elevation)
		cancelEnemyMend(e)
		e.Pose = "cast"
		e.PoseTime = .25
		e.Cooldown = math.Max(e.Cooldown, .5)
		return true
	}
	if e.HealCooldown > 0 || e.Cooldown > 0 || e.Windup > 0 || e.PoseTime > 0 || e.ChargeActive || e.ChargeRecovery > 0 || e.ReactMissTimer > 0 || !r.canStartEnemyAttack(e) {
		return false
	}
	target := -1
	ratio := 1.0
	for j := range r.Enemies {
		ally := &r.Enemies[j]
		if r.validMendTarget(e, ally) && ally.HP/ally.MaxHP < ratio {
			target = j
			ratio = ally.HP / ally.MaxHP
		}
	}
	if target < 0 {
		return false
	}
	e.HealTarget = r.Enemies[target].ID
	e.HealCooldown = 5
	e.Windup = .8
	e.AttackName = "Mend ally"
	e.Pose = "cast"
	if r.Enemies[target].X != e.X {
		e.Facing = math.Copysign(1, r.Enemies[target].X-e.X)
	}
	r.eventAtHeight("enemy_mend", e.X, e.Y-25, 0, e.Elevation)
	return true
}
