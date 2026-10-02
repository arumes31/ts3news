package rift

import "math"

// HazardSwitch is a deliberate guard-held interaction, never a loot-bearing prop.
type HazardSwitch struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Charge float64 `json:"charge"`
	Used   bool    `json:"used"`
}

func (r *Run) hazardSwitchTick(in Input, dt float64) {
	arena := r.Arena()
	s := arena.HazardSwitch
	if s == nil || s.Used || r.Status != "fighting" || r.Player.HP <= 0 {
		return
	}
	p := &r.Player
	busy := p.Cooldown > 0 || p.Knockdown > 0 || p.Pose == "attack" || p.Pose == "cast" || p.Pose == "hit" || p.Pose == "recovery" || p.Pose == "ultimate_anticipation" || p.Pose == "guard_break" || p.Pose == "dodge"
	target := Actor{X: s.X, Y: s.Y, Elevation: arena.Elevation(s.X, s.Y)}
	if !p.Guard || p.Jump > 0 || in.Attack || in.Skill != "" || in.Jump || in.Dodge || busy || math.Hypot(p.X-s.X, p.Y-s.Y) > 28 || p.Elevation != target.Elevation || !r.clearMeleePath(p, &target) {
		s.Charge = 0
		return
	}
	s.Charge = math.Min(.6, s.Charge+dt)
	if s.Charge < .6-1e-9 {
		return
	}
	s.Charge = .6
	s.Used = true
	for i := range arena.Hazards {
		arena.Hazards[i].Disabled = true
	}
	r.event("hazard_switch", s.X, s.Y, 0)
}
