package rift

import "math"

const missReactionSeconds = .24

// reactToMiss grants one nearby idle melee fighter a short approach, never a hit.
func (r *Run) reactToMiss() {
	selected := -1
	nearest := 180.0
	for i := range r.Enemies {
		e := &r.Enemies[i]
		if e.Kind != "goblin" && e.Kind != "knight" && e.Kind != "wolf" {
			continue
		}
		if e.HP <= 0 || e.Patrol && !e.Alerted || e.Windup > 0 || e.Cooldown > 0 || e.PoseTime > 0 || e.Knockdown > 0 || e.Jump > 0 || e.ChargeActive || e.ChargeRecovery > 0 || e.ReactMissTimer > 0 || e.isObjectiveProp() {
			continue
		}
		distance := math.Hypot(e.X-r.Player.X, e.Y-r.Player.Y)
		if distance > nearest || math.Abs(e.Y-r.Player.Y) > 60 || !r.clearMeleePath(e, &r.Player) || !r.canStartEnemyAttack(e) {
			continue
		}
		selected, nearest = i, distance
	}
	if selected < 0 {
		return
	}
	e := &r.Enemies[selected]
	e.ReactMissTimer = missReactionSeconds
	e.RouteX, e.RouteY = 0, 0
	r.eventAtHeight("miss_reaction", e.X, e.Y-25, 0, e.Elevation)
}

func (r *Run) tickMissReaction(e *Actor, dt float64) bool {
	if e.ReactMissTimer <= 0 {
		return false
	}
	e.ReactMissTimer = math.Max(0, math.Min(missReactionSeconds, e.ReactMissTimer)-dt)
	if e.ReactMissTimer < 1e-9 {
		e.ReactMissTimer = 0
	}
	if !r.clearMeleePath(e, &r.Player) {
		e.ReactMissTimer = 0
		return true
	}
	dx, dy := r.Player.X-e.X, r.Player.Y-e.Y
	if dx != 0 {
		e.Facing = math.Copysign(1, dx)
	}
	if math.Abs(dx) > 48 || math.Abs(dy) > 20 {
		speed := clamp(e.Speed, 80, 160) * 1.25
		r.moveActor(e, math.Copysign(math.Min(math.Max(0, math.Abs(dx)-48), speed*dt), dx), math.Copysign(math.Min(math.Abs(dy), speed*.6*dt), dy), false)
		e.Pose = "run"
	} else {
		e.Pose = "idle"
	}
	return true
}
