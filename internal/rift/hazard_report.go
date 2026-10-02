package rift

import (
	"fmt"
	"math"
	"sort"
)

// HazardUnsafeInterval is a half-open time interval with no sampled safe route.
// ActiveHazards contains zero-based indices in the authored arena hazard list.
type HazardUnsafeInterval struct {
	From          float64 `json:"from_seconds"`
	Until         float64 `json:"until_seconds"`
	ActiveHazards []int   `json:"active_hazards"`
}

// AuditHazardSafety checks entrance-to-exit walking access at every hazard phase
// transition in the requested window. It does not simulate a player's timed route.
func AuditHazardSafety(arena Arena, horizon float64) ([]HazardUnsafeInterval, error) {
	finite := func(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
	if !finite(horizon) || horizon <= 0 || horizon > 3600 {
		return nil, fmt.Errorf("hazard horizon must be in (0, 3600] seconds")
	}
	for i, h := range arena.Hazards {
		if h.Disabled {
			continue
		}
		for _, v := range []float64{h.X, h.Y, h.W, h.H, h.Period, h.Duration, h.Offset} {
			if !finite(v) {
				return nil, fmt.Errorf("hazard %d contains non-finite data", i)
			}
		}
		if h.W <= 0 || h.H <= 0 || h.Duration <= 0 || h.Period <= 1.2+h.Duration {
			return nil, fmt.Errorf("hazard %d has invalid geometry or no recovery interval", i)
		}
	}
	failures := []HazardUnsafeInterval{}
	if permanentHazardSafeRoute(arena) {
		return failures, nil
	}
	boundaries := []float64{0, horizon}
	for _, h := range arena.Hazards {
		if h.Disabled {
			continue
		}
		offset := h.Offset
		if offset >= 0 {
			offset = math.Mod(offset, h.Period)
		}
		for start := 1.2 - offset; start < horizon; start += h.Period {
			for _, edge := range []float64{start, start + h.Duration} {
				if edge > 0 && edge < horizon {
					boundaries = append(boundaries, edge)
				}
			}
		}
	}
	sort.Float64s(boundaries)
	unique := boundaries[:0]
	for _, edge := range boundaries {
		if len(unique) == 0 || edge != unique[len(unique)-1] {
			unique = append(unique, edge)
		}
	}
	cache := map[string]bool{}
	for i := 0; i+1 < len(unique); i++ {
		from, until := unique[i], unique[i+1]
		clock := from + (until-from)/2
		active := []int{}
		phaseArena := arena
		phaseArena.Hazards = nil
		for index, h := range arena.Hazards {
			phase := h.Phase(clock)
			if !h.Disabled && phase >= 1.2 && phase < 1.2+h.Duration {
				active = append(active, index)
				phaseArena.Hazards = append(phaseArena.Hazards, h)
			}
		}
		key := fmt.Sprint(active)
		safe, known := cache[key]
		if !known {
			safe = permanentHazardSafeRoute(phaseArena)
			cache[key] = safe
		}
		if !safe {
			failures = append(failures, HazardUnsafeInterval{From: from, Until: until, ActiveHazards: active})
		}
	}
	return failures, nil
}

// A permanent route outside every enabled hazard is safe for every combination
// of phases, including combinations that offsets normally keep apart.
func permanentHazardSafeRoute(arena Arena) bool {
	r := &Run{Level: &Level{Rooms: []Arena{arena}}}
	type point struct{ x, y float64 }
	start, goal := point{160, 410}, point{1450, 320}
	if arena.Entrance != nil { start = point{arena.Entrance.X, arena.Entrance.Y} }
	if arena.Exit != nil { goal = point{arena.Exit.X, arena.Exit.Y} }
	safe := func(x, y float64) bool {
		for _, h := range arena.Hazards {
			if !h.Disabled && contains(h.Obstacle, x, y, 12) {
				return false
			}
		}
		for _, o := range arena.solidObstacles() {
			if contains(o, x, y, 12) {
				return false
			}
		}
		return x >= 35 && x <= 1565 && y >= 315 && y <= 490 && arena.groundPath(x,y,x,y,12)
	}
	if !safe(float64(start.x), float64(start.y)) || !safe(float64(goal.x), float64(goal.y)) {
		return false
	}
	queue := []point{start}
	seen := map[point]bool{start: true}
	for head := 0; head < len(queue); head++ {
		at := queue[head]
		if math.Hypot(at.x-goal.x, at.y-goal.y) <= 14 {
			a := Actor{ID: "player", X: at.x, Y: at.y}
			dx,dy := (goal.x-at.x)/10,(goal.y-at.y)/10
			clear := true
			for part:=0;part<10;part++ {
				x,y:=a.X+dx,a.Y+dy
				if !safe(x,y) {clear=false;break}
				r.moveActor(&a,dx,dy,false)
				if math.Abs(a.X-x)>.001 || math.Abs(a.Y-y)>.001 {clear=false;break}
			}
			if clear {return true}
		}
		for _, step := range []point{{10, 0}, {-10, 0}, {0, 10}, {0, -10}} {
			next := point{at.x + step.x, at.y + step.y}
			if seen[next] || !safe(float64(next.x), float64(next.y)) {
				continue
			}
			a := Actor{ID: "player", X: float64(at.x), Y: float64(at.y)}
			clear := true
			// Check the path, not just the endpoints, and use actual collision/ledge rules.
			for part := 0; part < 5; part++ {
				x, y := a.X+float64(step.x)/5, a.Y+float64(step.y)/5
				if !safe(x, y) {
					clear = false
					break
				}
				r.moveActor(&a, float64(step.x)/5, float64(step.y)/5, false)
				if math.Abs(a.X-x) > .001 || math.Abs(a.Y-y) > .001 {
					clear = false
					break
				}
			}
			if clear {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	return false
}
