package rift

import "math"

// ReachabilityFailure identifies a target that cannot be approached on foot.
type ReachabilityFailure struct {
	ActorID string  `json:"actor_id"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Reason  string  `json:"reason"`
}

// AuditArenaReachability floods the campaign's 10-unit walking grid using the
// player's collision footprint. Targets must be within 16 units of a reached
// cell. Cover is treated as intact; jumping, hazards and abilities are excluded.
func AuditArenaReachability(arena Arena, start Actor, targets []Actor) []ReachabilityFailure {
	type cell struct{ x, y int }
	obstacles := arena.solidObstacles()
	clear := func(c cell) bool {
		if c.x < 4 || c.x > int(Width/10)-4 || c.y < 32 || c.y > 49 {
			return false
		}
		for _, o := range obstacles {
			if contains(o, float64(c.x*10), float64(c.y*10), 10) {
				return false
			}
		}
		return true
	}
	finite := func(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
	if !finite(start.X) || !finite(start.Y) || start.X < 40 || start.X > Width-40 || start.Y < 315 || start.Y > 490 {
		return []ReachabilityFailure{{ActorID: start.ID, X: start.X, Y: start.Y, Reason: "invalid_start"}}
	}
	first := cell{int(math.Round(start.X / 10)), int(math.Round(start.Y / 10))}
	if !clear(first) {
		return []ReachabilityFailure{{ActorID: start.ID, X: start.X, Y: start.Y, Reason: "blocked_start"}}
	}
	queue := []cell{first}
	seen := map[cell]bool{first: true}
	for head := 0; head < len(queue); head++ {
		at := queue[head]
		for _, step := range []cell{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			next := cell{at.x + step.x, at.y + step.y}
			if !seen[next] && clear(next) {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	failures := []ReachabilityFailure{}
	for _, target := range targets {
		reached := false
		for _, at := range queue {
			if math.Hypot(float64(at.x*10)-target.X, float64(at.y*10)-target.Y) < 16 {
				reached = true
				break
			}
		}
		if !reached {
			failures = append(failures, ReachabilityFailure{ActorID: target.ID, X: target.X, Y: target.Y, Reason: "unreachable"})
		}
	}
	return failures
}
