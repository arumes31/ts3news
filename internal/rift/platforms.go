package rift

import "math"

// RaisedPlatform is a walkable surface with a sloped approach on every edge.
// Elevation is separate from jump height and does not grant airborne immunity.
type RaisedPlatform struct {
	Obstacle
	ID    string  `json:"id"`
	Rise  float64 `json:"rise"`
	Ramp  float64 `json:"ramp"`
	Floor string  `json:"floor"`
}

func (p RaisedPlatform) elevation(x, y float64) float64 {
	if p.Rise <= 0 || p.Ramp <= 0 || p.Rise > p.Ramp || p.Ramp*2 > math.Min(p.W, p.H) {
		return 0
	}
	distance := math.Min(math.Min(x-p.X, p.X+p.W-x), math.Min(y-p.Y, p.Y+p.H-y))
	return p.Rise * clamp(distance/p.Ramp, 0, 1)
}

func (a Arena) surfaceAt(x, y float64) (float64, string) {
	height, floor := 0.0, ""
	for _, p := range a.Platforms {
		if candidate := p.elevation(x, y); candidate > height {
			height, floor = candidate, p.Floor
		}
	}
	return height, floor
}

func (a Arena) Elevation(x, y float64) float64 {
	height, _ := a.surfaceAt(x, y)
	return height
}
