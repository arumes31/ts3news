package rift

import "math"

// RoundArena describes a circular floor in the game's compressed depth view.
// Radii include the visible edge; actor clearance reserves room inside it.
type RoundArena struct {
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	RadiusX float64 `json:"radius_x"`
	RadiusY float64 `json:"radius_y"`
}

func (a RoundArena) containsGround(x, y, radius float64) bool {
	rx, ry := a.RadiusX-radius, a.RadiusY-radius
	if rx <= 0 || ry <= 0 {
		return false
	}
	dx, dy := (x-a.X)/rx, (y-a.Y)/ry
	return dx*dx+dy*dy <= 1+1e-12
}

func (a RoundArena) valid() bool {
	for _, v := range []float64{a.X, a.Y, a.RadiusX, a.RadiusY} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return a.RadiusX >= 300 && a.RadiusY >= 70 && a.X-a.RadiusX >= 35 && a.X+a.RadiusX <= Width-35 && a.Y-a.RadiusY >= 315 && a.Y+a.RadiusY <= 490
}

// ValidRound checks saved floor geometry and explicit entry/exit anchors.
func (a Arena) ValidRound() bool {
	if a.Round == nil {
		return true
	}
	if !a.Round.valid() {
		return false
	}
	for _, point := range []*ArenaEntrance{a.Entrance, a.Exit} {
		if point != nil && !a.Round.containsGround(point.X, point.Y, 10) {
			return false
		}
	}
	return true
}
