package rift

import "math"

// Pull moves at most 24 units per contact, preserving collision clearance and
// declining ledge snaps that would turn a short pull into a long displacement.
func (r *Run) pullIntoVoidWell(h Hazard) {
	dx, dy := (h.X+h.W/2-r.Player.X)*.3, (h.Y+h.H/2-r.Player.Y)*.3
	distance := math.Hypot(dx, dy)
	if distance == 0 {
		return
	}
	if distance > 24 {
		dx *= 24 / distance
		dy *= 24 / distance
		distance = 24
	}
	steps := int(math.Ceil(distance / 4))
	dx /= float64(steps)
	dy /= float64(steps)
	for i := 0; i < steps; i++ {
		next := r.Player
		r.moveActor(&next, dx, dy, false)
		if math.Hypot(next.X-r.Player.X, next.Y-r.Player.Y) > math.Hypot(dx, dy)+1e-8 {
			break
		}
		r.Player = next
	}
	r.Floor = r.FloorMaterial()
}
