package rift

import "math"

// settleEnemySpawn keeps the preferred point when safe, otherwise chooses the
// nearest clear ground point. Avoid whole enabled hazard footprints so a warning
// phase cannot put a newly arriving enemy inside the next pulse.
// Authored campaign arenas retain clear ground; false reports an invalid arena
// with no safe candidate without silently removing its enemies or hazards.
func (arena Arena) settleEnemySpawn(a *Actor) bool {
	radius := actorClearance(a)
	walls := arena.solidObstacles()
	safe := func(x, y float64) bool {
		if x < 35 || x > Width-35 || y < 315 || y > 490 {
			return false
		}
		for _, o := range walls {
			if contains(o, x, y, radius) {
				return false
			}
		}
		for _, h := range arena.Hazards {
			if !h.Disabled && contains(h.Obstacle, x, y, radius) {
				return false
			}
		}
		for _, edge := range arena.DropEdges {
			if x >= edge.X-radius && x <= edge.X+edge.W+radius && y > edge.Y-radius && y < edge.LandingY+radius {
				return false
			}
		}
		return true
	}
	if safe(a.X, a.Y) {
		return true
	}
	best := math.Inf(1)
	bestX, bestY := a.X, a.Y
	for x := 35.0; x <= Width-35; x += 5 {
		for y := 315.0; y <= 490; y += 5 {
			distance := (x-a.X)*(x-a.X) + (y-a.Y)*(y-a.Y)
			if distance >= best || !safe(x, y) {
				continue
			}
			best, bestX, bestY = distance, x, y
		}
	}
	if math.IsInf(best, 1) {
		return false
	}
	a.X, a.Y = bestX, bestY
	return true
}
