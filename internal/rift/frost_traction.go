package rift

import "math"

// Active frost uses saved actor velocity. Unflagged frozen arenas retain their
// original movement, while guard, dodge and successful jumps keep direct control.
func (r *Run) movePlayerWithTraction(in Input, vx, vy, dt float64) {
	p := &r.Player
	jumpReady := in.Jump && p.Jump == 0 && r.SkillTimers["jump"] == 0 && p.Pose != "recovery" && p.Pose != "ultimate_anticipation"
	slippery := false
	if r.Status == "fighting" && p.HP > 0 && p.Jump == 0 && !jumpReady && !p.Guard && p.Pose != "dodge" && p.Knockdown == 0 {
		for _, h := range r.Arena().Hazards {
			phase := h.Phase(r.Clock)
			if h.Slippery && h.Kind == "ice" && !h.Disabled && phase >= 1.2 && phase < 1.2+h.Duration && contains(h.Obstacle, p.X, p.Y, 0) {
				slippery = true
				break
			}
		}
	}
	if slippery {
		vx = p.Vx + clamp(vx-p.Vx, -600*dt, 600*dt)
		vy = p.Vy + clamp(vy-p.Vy, -360*dt, 360*dt)
	}
	beforeX, beforeY := p.X, p.Y
	r.moveActor(p, vx*dt, vy*dt, false)
	// A wall or arena edge consumes momentum rather than storing a hidden slide.
	if slippery && dt > 0 {
		if math.Abs(p.X-beforeX-vx*dt) > 1e-7 {
			vx = 0
		}
		if math.Abs(p.Y-beforeY-vy*dt) > 1e-7 {
			vy = 0
		}
	}
	if !slippery && in.X == 0 && in.Y == 0 {
		vx, vy = 0, 0
	}
	p.Vx, p.Vy = vx, vy
}
