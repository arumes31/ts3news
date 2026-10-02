package rift

import (
	"math"
	"strings"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

func canonicalBurrower(m content.Mob) bool {
	// The shared noun pool keeps rat identity aligned with the active locale.
	noun := "Rat"
	if nouns := i18n.Pool("pool.mob.noun"); len(nouns) > 0 {
		noun = nouns[0]
	}
	return m.Type == content.MobCommon && strings.HasSuffix(m.Name, " "+noun)
}

func (r *Run) cancelEnemyBurrow(e *Actor) {
	if !e.Burrowed {
		return
	}
	e.Burrowed = false
	e.Windup = 0
	e.AttackName = ""
	e.Cooldown = math.Max(e.Cooldown, 2)
	r.eventAtHeight("burrow_cancel", e.X, e.Y-25, 0, e.Elevation)
}

// Burrowing follows a locked, walkable route. The final second is stationary:
// the visible emergence area never follows the player after the cue starts.
func (r *Run) tickEnemyBurrow(e *Actor, dt float64) bool {
	if !e.Burrowing || e.Kind == "boss" || e.Kind == "archer" || e.Kind == "treasure" || e.isObjectiveProp() {
		return false
	}
	if e.BurrowRecovery > 0 {
		e.BurrowRecovery = math.Max(0, math.Min(.9, e.BurrowRecovery)-dt)
		if e.PoseTime == 0 {
			e.Pose = "recovery"
		}
		return true
	}
	if e.Burrowed {
		travel := math.Min(dt, math.Max(0, e.Windup-1))
		remaining := 240 * travel
		for remaining > 0 {
			dx, dy := e.TargetX-e.X, e.TargetY-e.Y
			distance := math.Hypot(dx, dy)
			if distance < .01 {
				break
			}
			step := math.Min(6, math.Min(distance, remaining))
			sx, sy := dx/distance*step, dy/distance*step
			x, y := e.X, e.Y
			r.moveActor(e, sx, sy, false)
			if math.Abs(e.X-x-sx) > .01 || math.Abs(e.Y-y-sy) > .01 {
				r.cancelEnemyBurrow(e)
				e.Pose = "idle"
				return true
			}
			remaining -= step
		}
		e.Windup = math.Max(0, math.Min(2, e.Windup)-dt)
		if e.Windup > 1e-9 {
			e.Pose = "windup"
			return true
		}
		e.Windup = 0
		e.Burrowed = false
		e.AttackName = ""
		e.Attacks++
		e.BurrowRecovery = .9
		e.Cooldown = 2
		e.Pose = "attack"
		e.PoseTime = .35
		r.eventAtHeight("burrow_emerge", e.X, e.Y, 0, e.Elevation)
		dx, dy := (r.Player.X-e.X)/75, (r.Player.Y-e.Y)/40
		if dx*dx+dy*dy <= 1 && r.clearMeleePath(e, &r.Player) {
			if r.Player.Jump >= .1 || r.SkillTimers["dodge_invulnerability"] > 0 {
				r.recordDodge()
			} else {
				r.hurtPlayerFromEnemy(math.Max(20, e.Damage), e.X, e.Y, e.ID)
			}
		}
		return true
	}
	distance := math.Hypot(r.Player.X-e.X, r.Player.Y-e.Y)
	if e.Cooldown > 0 || e.Windup > 0 || e.PoseTime > 0 || e.Jump > 0 || e.Elevation != 0 || r.Player.Elevation != 0 || distance < 100 || distance > 240 || !r.clearPursuitPath(e, &r.Player) || !r.canStartEnemyAttack(e) {
		return false
	}
	e.Burrowed = true
	e.Windup = distance/240 + 1
	e.AttackName = "Burrow"
	e.Pose = "windup"
	e.TargetX, e.TargetY = r.Player.X, r.Player.Y
	e.Facing = math.Copysign(1, e.TargetX-e.X)
	if e.Pack {
		r.PackAttackLockout = .45
		r.Stats.PackAttacks++
	}
	r.eventAtHeight("burrow_start", e.X, e.Y-25, 0, e.Elevation)
	return true
}
