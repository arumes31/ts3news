package rift

import (
	"fmt"
	"math"
)

// WindGust adds a fixed horizontal drift inside an authored corridor. It never
// changes a projectile's stored launch velocity, damage, ownership or lifetime.
type WindGust struct {
	Obstacle
	Period   float64 `json:"period"`
	Offset   float64 `json:"offset"`
	Duration float64 `json:"duration"`
	VX       float64 `json:"vx"`
}

func (w WindGust) Phase(clock float64) float64 { return math.Mod(clock+w.Offset, w.Period) }

func (r *Run) projectileWind(x, y float64) float64 {
	if r.Status != "fighting" {
		return 0
	}
	drift := 0.0
	for _, w := range r.Arena().WindGusts {
		phase := w.Phase(r.Clock)
		if phase >= 1.2 && phase < 1.2+w.Duration && contains(w.Obstacle, x, y, 0) {
			drift += w.VX
		}
	}
	return clamp(drift, -80, 80)
}

func (r *Run) windGustTick() {
	if r.Status != "fighting" || r.Player.HP <= 0 {
		return
	}
	if r.SkillTimers == nil {
		r.SkillTimers = map[string]float64{}
	}
	for i, w := range r.Arena().WindGusts {
		phase := w.Phase(r.Clock)
		warnKey := fmt.Sprintf("wind-warn-%d", i)
		activeKey := fmt.Sprintf("wind-active-%d", i)
		if phase < 1.2 && r.SkillTimers[warnKey] <= 0 {
			r.SkillTimers[warnKey] = w.Period - phase + .05
			r.event("wind_warning", w.X+w.W/2, w.Y+w.H/2, 0)
		}
		if phase >= 1.2 && phase < 1.2+w.Duration && r.SkillTimers[activeKey] <= 0 {
			r.SkillTimers[activeKey] = w.Period - phase + .05
			r.event("wind_gust", w.X+w.W/2, w.Y+w.H/2, 0)
		}
	}
}
