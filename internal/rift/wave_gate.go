package rift

import "math"

// WaveGate is a frozen threshold and its saved, combat-clock-driven state.
// A closed gate is tall cover; both authored bypasses remain open.
type WaveGate struct {
	Obstacle
	Closed  bool    `json:"closed"`
	CloseIn float64 `json:"close_in"`
}

func (r *Run) warnWaveGate() {
	g := r.RoomObjective.Gate
	if g == nil {
		return
	}
	g.Closed = false
	g.CloseIn = .8
	r.event("wave_gate_warning", g.X+g.W/2, g.Y+g.H/2, .8)
}

func (r *Run) openWaveGate() {
	g := r.RoomObjective.Gate
	if g == nil {
		return
	}
	if g.Closed {
		r.event("wave_gate_open", g.X+g.W/2, g.Y+g.H/2, 0)
	}
	g.Closed = false
	g.CloseIn = 0
}

func (r *Run) tickWaveGate(dt float64) {
	g := r.RoomObjective.Gate
	if g == nil || g.Closed {
		return
	}
	g.CloseIn = math.Max(0, g.CloseIn-dt)
	if g.CloseIn > 1e-9 {
		return
	}
	g.CloseIn = 0
	occupied := func(a *Actor) bool { return a.HP > 0 && contains(g.Obstacle, a.X, a.Y, actorClearance(a)+1) }
	if occupied(&r.Player) {
		return
	}
	for i := range r.Enemies {
		if occupied(&r.Enemies[i]) {
			return
		}
	}
	g.Closed = true
	r.event("wave_gate_close", g.X+g.W/2, g.Y+g.H/2, 0)
}

// Check the swept footprint as well as movement endpoints: a dodge can span
// the complete gate width in one tick. A tiny inset permits moving away from
// a touching boundary without making the actor stick to it.
func (r *Run) waveGateBlocksPath(x1, y1, x2, y2, radius float64) bool {
	o := r.RoomObjective
	if o == nil || o.Kind != "survive_waves" || o.Gate == nil || !o.Gate.Closed || r.Practice != nil {
		return false
	}
	g := o.Gate
	box := Obstacle{g.X - radius + .001, g.Y - radius + .001, g.W + 2*radius - .002, g.H + 2*radius - .002}
	_, hit := obstacleImpact(x1, y1, x2, y2, box)
	return hit
}
