package rift

import "math"

// WaterCurrent is a steady ground-level flow. Its velocity and footprint are
// frozen with the arena; actor input velocity remains under player/AI control.
type WaterCurrent struct {
	Obstacle
	VX float64 `json:"vx"`
	VY float64 `json:"vy"`
}

func (r *Run) currentVelocity(a *Actor) (float64, float64) {
	if a.HP <= 0 || a.Jump > 0 || a.Elevation > 0 || a.Knockdown > 0 || a.Flying || a.Burrowed || a.isObjectiveProp() || a == &r.Player && (a.Guard || a.Pose == "dodge") {
		return 0, 0
	}
	vx, vy := 0.0, 0.0
	for _, c := range r.Arena().WaterCurrents {
		if contains(c.Obstacle, a.X, a.Y, 0) {
			vx += c.VX
			vy += c.VY
		}
	}
	if speed := math.Hypot(vx, vy); speed > 40 {
		vx *= 40 / speed
		vy *= 40 / speed
	}
	return vx, vy
}

func (r *Run) waterCurrentTick(dt float64) {
	if r.Status != "fighting" || r.Player.HP <= 0 {
		return
	}
	if len(r.Arena().WaterCurrents) == 0 {
		delete(r.SkillTimers, "water_current_contact")
		return
	}
	vx, vy := r.currentVelocity(&r.Player)
	if vx != 0 || vy != 0 {
		if r.SkillTimers == nil {
			r.SkillTimers = map[string]float64{}
		}
		if r.SkillTimers["water_current_contact"] <= 0 {
			r.event("water_current", r.Player.X, r.Player.Y, 0)
		}
		r.SkillTimers["water_current_contact"] = 1
		r.moveActor(&r.Player, vx*dt, vy*dt, false)
	} else {
		delete(r.SkillTimers, "water_current_contact")
	}
	for i := range r.Enemies {
		a := &r.Enemies[i]
		vx, vy := r.currentVelocity(a)
		if vx != 0 || vy != 0 {
			r.moveActor(a, vx*dt, vy*dt, false)
		}
	}
}
