package rift

import "math"

// DropEdge marks a one-way descent; landing coordinates are frozen in the arena.
type DropEdge struct {
	ID       string  `json:"id"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	W        float64 `json:"w"`
	LandingY float64 `json:"landing_y"`
}

func (r *Run) safeDropLanding(a *Actor, x, y float64) bool {
	if x < 35 || x > Width-35 || y < 315 || y > 490 {
		return false
	}
	radius := actorClearance(a)
	if !r.Arena().groundPath(x, y, x, y, radius) { return false }
	for _, o := range r.Arena().solidObstacles() {
		if contains(o, x, y, radius) {
			return false
		}
	}
	for _, h := range r.Arena().Hazards {
		if !h.Disabled && contains(h.Obstacle, x, y, radius) {
			return false
		}
	}
	return true
}

// Ledge navigation takes ascending pursuers around an end and sideways pursuers
// below the face. Saved detours finish before ordinary target tracking resumes.
func (r *Run) navigateDropEdge(a *Actor, dx, dy float64) (float64, float64) {
	step := math.Max(math.Abs(dx), math.Abs(dy))
	for _, edge := range r.Arena().DropEdges {
		if a.LedgeRoute == edge.ID {
			if a.LedgeRouteSide {
				// Finish the lower-floor bypass before tracking the target again.
				if a.Y < edge.LandingY {
					return 0, math.Min(step, edge.LandingY-a.Y)
				}
				if math.Abs(a.X-a.LedgeRouteX) > 1 {
					return math.Copysign(math.Min(step, math.Abs(a.X-a.LedgeRouteX)), a.LedgeRouteX-a.X), 0
				}
				a.LedgeRoute, a.LedgeRouteSide = "", false
				return dx, dy
			}
			if dy >= 0 {
				a.LedgeRoute = ""
				return dx, dy
			}
			if a.Y < edge.Y-actorClearance(a)-2 {
				a.LedgeRoute = ""
				return dx, dy
			}
			if math.Abs(a.X-a.LedgeRouteX) > 1 {
				return math.Copysign(math.Min(step, math.Abs(a.X-a.LedgeRouteX)), a.LedgeRouteX-a.X), 0
			}
			return 0, -step
		}
	}
	a.LedgeRoute, a.LedgeRouteSide = "", false
	for _, edge := range r.Arena().DropEdges {
		if a.Y > edge.Y && a.Y < edge.LandingY && crossesDropFaceX(edge, a.X, a.X+dx) {
			a.LedgeRoute, a.LedgeRouteSide = edge.ID, true
			a.LedgeRouteX = edge.X + edge.W + actorClearance(a) + 8
			if dx < 0 {
				a.LedgeRouteX = edge.X - actorClearance(a) - 8
			}
			return 0, math.Min(step, edge.LandingY-a.Y)
		}
	}
	if dy >= 0 {
		return dx, dy
	}
	for _, edge := range r.Arena().DropEdges {
		x := a.X + dx
		if x < edge.X || x > edge.X+edge.W || a.Y < edge.LandingY || a.Y+dy >= edge.LandingY {
			continue
		}
		a.LedgeRoute = edge.ID
		a.LedgeRouteX = edge.X - actorClearance(a) - 8
		if a.X-edge.X > edge.X+edge.W-a.X {
			a.LedgeRouteX = edge.X + edge.W + actorClearance(a) + 8
		}
		return math.Copysign(step, a.LedgeRouteX-a.X), 0
	}
	return dx, dy
}

func crossesDropFaceX(edge DropEdge, from, to float64) bool {
	return math.Max(from, to) >= edge.X && math.Min(from, to) <= edge.X+edge.W
}

func (r *Run) dropFaceBlocksX(a *Actor, x float64) bool {
	for _, edge := range r.Arena().DropEdges {
		if a.Y > edge.Y && a.Y < edge.LandingY && crossesDropFaceX(edge, a.X, x) {
			return true
		}
	}
	return false
}

func (r *Run) crossDropEdge(a *Actor, nextY float64) float64 {
	for _, edge := range r.Arena().DropEdges {
		if a.X < edge.X || a.X > edge.X+edge.W {
			continue
		}
		if a.Y >= edge.LandingY && nextY < edge.LandingY || a.Y > edge.Y && nextY <= edge.Y {
			return a.Y
		}
		if a.Y <= edge.Y && nextY > edge.Y {
			if edge.LandingY <= edge.Y || !r.safeDropLanding(a, a.X, edge.LandingY) {
				return a.Y
			}
			a.Jump = 0
			if a.Pose == "idle" || a.Pose == "run" || a.Pose == "jump" {
				a.Pose, a.PoseTime = "land", .14
			}
			if a.ID == r.Player.ID {
				r.event("ledge_drop", a.X, edge.LandingY, edge.LandingY-edge.Y)
				r.event("land", a.X, edge.LandingY, .65)
			}
			return edge.LandingY
		}
	}
	return nextY
}
