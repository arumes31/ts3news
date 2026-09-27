package rift

import (
	"math"
	"strings"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

func canonicalDiver(m content.Mob) bool {
	noun := "Bat"
	if nouns := i18n.Pool("pool.mob.noun"); len(nouns) > 7 {
		noun = nouns[7]
	}
	return m.Type == content.MobCommon && strings.HasSuffix(m.Name, " "+noun)
}

func recoverEnemyDive(e *Actor) {
	e.Diving = false
	e.Windup = 0
	e.AttackName = ""
	e.DiveRecovery = 1
	e.Cooldown = math.Max(e.Cooldown, 2.2)
	e.Pose = "recovery"
	e.PoseTime = 0
	e.RouteX, e.RouteY = 0, 0
}

func (r *Run) finishEnemyDive(e *Actor) {
	recoverEnemyDive(e)
	r.eventAtHeight("dive_land", e.X, e.Y-25, 0, e.Elevation)
}

func (r *Run) interruptEnemyDive(e *Actor) {
	if e.Diving || e.AttackName == "Dive" {
		pose, poseTime := e.Pose, e.PoseTime
		recoverEnemyDive(e)
		e.Pose, e.PoseTime = pose, poseTime
		r.eventAtHeight("dive_cancel", e.X, e.Y-25, 0, e.Elevation)
	}
}

// A low swoop remains hittable, uses the saved ground route and never tunnels
// through cover. Its warning occupies an ordinary attack slot until landing.
func (r *Run) tickEnemyDive(e *Actor, dt float64) bool {
	if !e.Flying || e.Kind == "boss" || e.Kind == "archer" || e.Kind == "treasure" || e.isObjectiveProp() {
		return false
	}
	if e.DiveRecovery > 0 {
		e.DiveRecovery = math.Max(0, math.Min(1, e.DiveRecovery)-dt)
		if e.PoseTime == 0 {
			e.Pose = "recovery"
		}
		return true
	}
	if e.AttackName == "Dive" && !e.Diving {
		if e.Windup <= 0 {
			r.finishEnemyDive(e)
			return true
		}
		e.Windup = math.Max(0, e.Windup-dt)
		if e.Windup <= 1e-9 {
			e.Windup = 0
			e.Diving = true
			e.Attacks++
			e.Pose = "attack"
			e.PoseTime = math.Hypot(e.TargetX-e.X, e.TargetY-e.Y)/300 + .05
			r.eventAtHeight("dive_swoop", e.X, e.Y-25, 0, e.Elevation)
		}
		return true
	}
	if e.Diving {
		remaining := 300 * dt
		for remaining > 0 {
			dx, dy := e.TargetX-e.X, e.TargetY-e.Y
			distance := math.Hypot(dx, dy)
			if distance < 1 {
				r.finishEnemyDive(e)
				return true
			}
			step := math.Min(6, math.Min(distance, remaining))
			sx, sy := dx/distance*step, dy/distance*step
			x, y := e.X, e.Y
			r.moveActor(e, sx, sy, false)
			if math.Abs(e.X-x-sx) > .01 || math.Abs(e.Y-y-sy) > .01 {
				r.finishEnemyDive(e)
				return true
			}
			remaining -= step
			if math.Abs(r.Player.X-e.X) < 30 && math.Abs(r.Player.Y-e.Y) < 20 && r.clearMeleePath(e, &r.Player) {
				if r.Player.Jump >= .2 || r.SkillTimers["dodge_invulnerability"] > 0 {
					r.recordDodge()
				} else {
					r.hurtPlayerFromEnemy(math.Max(19, e.Damage), e.X, e.Y, e.ID)
				}
				r.finishEnemyDive(e)
				return true
			}
		}
		return true
	}
	distance := math.Hypot(r.Player.X-e.X, r.Player.Y-e.Y)
	if e.Cooldown > 0 || e.Windup > 0 || e.PoseTime > 0 || e.Jump > 0 || e.Elevation != 0 || r.Player.Elevation != 0 || distance < 110 || distance > 300 || math.Abs(r.Player.Y-e.Y) > 100 || !r.clearPursuitPath(e, &r.Player) || !r.canStartEnemyAttack(e) {
		return false
	}
	e.TargetX, e.TargetY = r.Player.X, r.Player.Y
	e.Facing = math.Copysign(1, e.TargetX-e.X)
	e.Windup = 1
	e.AttackName = "Dive"
	e.Pose = "windup"
	if e.Pack {
		r.PackAttackLockout = .45
		r.Stats.PackAttacks++
	}
	r.eventAtHeight("dive_warning", e.X, e.Y-25, 0, e.Elevation)
	return true
}
