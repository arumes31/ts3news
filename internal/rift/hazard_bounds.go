package rift

import "math"

// ContactBounds returns the damaging footprint at the saved simulation clock.
// The authored obstacle remains the full warning envelope and spawn exclusion.
func (h Hazard) ContactBounds(clock float64) Obstacle {
	box := h.Obstacle
	if h.Kind == "sweeping_flame" && h.Duration > 0 {
		width := min(36.0, box.W)
		progress := clamp((h.Phase(clock)-1.2)/h.Duration, 0, 1)
		box.X += (box.W - width) * progress
		box.W = width
	}
	if h.Kind == "rotating_blade" && h.Duration > 0 {
		w, height := min(20.0, box.W), min(20.0, box.H)
		angle := clamp((h.Phase(clock)-1.2)/h.Duration, 0, 1) * 2 * math.Pi
		box.X += (box.W - w) / 2 * (1 + math.Cos(angle))
		box.Y += (box.H - height) / 2 * (1 + math.Sin(angle))
		box.W, box.H = w, height
	}
	if h.Kind == "moving_poison" && h.Duration > 0 {
		w := min(80.0, box.W)
		progress := clamp((h.Phase(clock)-1.2)/h.Duration, 0, 1)
		box.X += (box.W - w) * (1 - math.Abs(2*progress-1))
		box.W = w
	}
	return box
}
