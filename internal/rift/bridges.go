package rift

import "math"

// NarrowBridge saves a deck across an X span. The areas above and below the
// deck are not walkable ground, but remain open to projectiles.
type NarrowBridge struct {
	Obstacle
	ID string `json:"id"`
}

// groundPath checks the whole ground trajectory, including large impulses.
// Jump height does not change where an actor may land on a railed bridge.
func (a Arena) groundPath(x1, y1, x2, y2, radius float64) bool {
	for _, b := range a.Bridges {
		left, right := b.X-radius, b.X+b.W+radius
		enter, leave := 0.0, 1.0
		if x1 == x2 {
			if x1 <= left || x1 >= right {
				continue
			}
		} else {
			enter, leave = (left-x1)/(x2-x1), (right-x1)/(x2-x1)
			if enter > leave {
				enter, leave = leave, enter
			}
			enter, leave = math.Max(0, enter), math.Min(1, leave)
			if enter >= leave {
				continue
			}
		}
		from, to := y1+(y2-y1)*enter, y1+(y2-y1)*leave
		if math.Min(from, to) < b.Y+radius || math.Max(from, to) > b.Y+b.H-radius {
			return false
		}
	}
	return true
}

// bridgeApproach aligns pursuers with a deck before crossing its boundary.
// Once on the deck, lateral pursuit stays within the rails until the exit.
func (a Arena) bridgeApproach(actor *Actor, dx, dy float64) (float64, float64) {
	radius := actorClearance(actor)
	for _, b := range a.Bridges {
		left, right := b.X-radius, b.X+b.W+radius
		if math.Max(actor.X, actor.X+dx) <= left || math.Min(actor.X, actor.X+dx) >= right {
			continue
		}
		low, high := b.Y+radius, b.Y+b.H-radius
		if actor.Y < low || actor.Y > high {
			step := math.Max(math.Abs(dx), math.Abs(dy))
			return 0, clamp(b.Y+b.H/2-actor.Y, -step, step)
		}
		// Keep the approach on the deck even when the chased target is beyond a rail.
		dy = clamp(actor.Y+dy, low, high) - actor.Y
	}
	return dx, dy
}

// ValidBridges bounds saved geometry and leaves an approach between separate decks.
func (a Arena) ValidBridges() bool {
	if len(a.Bridges) > 4 {
		return false
	}
	ids := map[string]bool{}
	for i, b := range a.Bridges {
		if b.ID == "" || len(b.ID) > 80 || ids[b.ID] {
			return false
		}
		ids[b.ID] = true
		for _, v := range []float64{b.X, b.Y, b.W, b.H} {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return false
			}
		}
		if b.X < 100 || b.W < 80 || b.X+b.W > Width-100 || b.Y < 315 || b.H < 80 || b.Y+b.H > 490 {
			return false
		}
		for _, other := range a.Bridges[:i] {
			if b.X < other.X+other.W+80 && other.X < b.X+b.W+80 {
				return false
			}
		}
	}
	return true
}
