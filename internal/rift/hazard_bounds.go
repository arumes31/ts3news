package rift

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
	return box
}
