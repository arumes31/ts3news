package rift

import "math"

// tickSupportRetreat seeks reachable ground behind a living front-line ally.
// Cover is recomputed against the current player side, never a stale saved point.
func (r *Run) tickSupportRetreat(e *Actor, dt float64, previousCover string) bool {
	if !e.Support || e.Kind == "boss" || e.Kind == "treasure" || e.Windup > 0 || e.PoseTime > 0 || e.Knockdown > 0 || e.Jump > 0 || e.ChargeActive || e.ChargeRecovery > 0 || e.ReactMissTimer > 0 {
		return false
	}
	p := &r.Player
	if math.Hypot(e.X-p.X, e.Y-p.Y) > 240 && previousCover == "" {
		return false
	}
	selected := -1
	best := math.Inf(1)
	var target Actor
	for i := range r.Enemies {
		defender := &r.Enemies[i]
		if defender == e || defender.HP <= 0 || defender.isObjectiveProp() || defender.Kind == "boss" || defender.Kind == "treasure" || defender.Kind != "knight" && !defender.Shield || defender.Knockdown > 0 || defender.Jump > 0 || defender.ChargeActive || defender.Pose == "stagger" && defender.PoseTime > 0 || defender.Patrol && !defender.Alerted || math.Hypot(defender.X-e.X, defender.Y-e.Y) > 220 {
			continue
		}
		dx, dy := defender.X-p.X, defender.Y-p.Y
		distance := math.Hypot(dx, dy)
		if distance < 30 {
			continue
		}
		ux, uy := dx/distance, dy/distance
		behind := (e.X-defender.X)*ux + (e.Y-defender.Y)*uy
		across := math.Abs((e.X-defender.X)*uy - (e.Y-defender.Y)*ux)
		if behind >= 60 && across <= 55 && r.clearPursuitPath(e, defender) {
			return false
		}
		goal := Actor{ID: e.ID, Kind: e.Kind, X: clamp(defender.X+ux*90, 35, 1565), Y: clamp(defender.Y+uy*90, 315, 490)}
		if (goal.X-defender.X)*ux+(goal.Y-defender.Y)*uy < 45 || (goal.X-e.X)*(e.X-p.X)+(goal.Y-e.Y)*(e.Y-p.Y) < 0 || !r.clearPursuitPath(e, &goal) {
			continue
		}
		travel := math.Hypot(goal.X-e.X, goal.Y-e.Y)
		if travel < best {
			selected, best, target = i, travel, goal
		}
	}
	if selected < 0 || best < 1 {
		return false
	}
	speed := clamp(e.Speed, 80, 140)
	step := math.Min(best, speed*dt)
	beforeX, beforeY := e.X, e.Y
	r.moveActor(e, (target.X-e.X)/best*step, (target.Y-e.Y)/best*step, false)
	if math.Hypot(e.X-beforeX, e.Y-beforeY) < .01 {
		return false
	}
	e.SupportCoverID = r.Enemies[selected].ID
	e.Pose = "run"
	if p.X != e.X {
		e.Facing = math.Copysign(1, p.X-e.X)
	}
	if previousCover != e.SupportCoverID {
		r.eventAtHeight("support_retreat", e.X, e.Y-25, 0, e.Elevation)
	}
	return true
}
