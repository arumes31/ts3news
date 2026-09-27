package bot

import (
	"errors"
	"math"

	"ts3news/internal/rift"
)

func validateRiftHazardTiming(run *rift.Run) error {
	invalid := errors.New("invalid rift snapshot hazard timing")
	// Keep submillisecond precision when adding simulation steps or phase offsets.
	const clockLimit = 1 << 40
	if !(run.Clock >= 0 && run.Clock <= clockLimit) {
		return invalid
	}
	validArena := func(arena rift.Arena) bool {
		if len(arena.WaterCurrents) > 4 {
			return false
		}
		for _, c := range arena.WaterCurrents {
			if !(math.Hypot(c.VX, c.VY) > 0 && math.Hypot(c.VX, c.VY) <= 40 && c.X >= 35 && c.Y >= 315 && c.W > 0 && c.H > 0 && c.X+c.W <= 1565 && c.Y+c.H <= 490) {
				return false
			}
		}
		if len(arena.WindGusts) > 4 {
			return false
		}
		for _, w := range arena.WindGusts {
			if !(w.Period > 1.2 && w.Period <= 30 && w.Duration > 0 && 1.2+w.Duration < w.Period && w.Offset >= 0 && w.Offset <= clockLimit && w.VX >= -80 && w.VX <= 80 && w.VX != 0 && w.X >= 35 && w.Y >= 315 && w.W > 0 && w.H > 0 && w.X+w.W <= 1565 && w.Y+w.H <= 490) {
				return false
			}
		}
		for _, h := range arena.Hazards {
			// Authored cycles are at most seven seconds. Thirty seconds leaves content
			// headroom while keeping warning-repeat timers inside the saved timer bound.
			if !(h.Period > 1.2 && h.Period <= 30 && h.Duration > 0 && 1.2+h.Duration < h.Period && h.Offset >= 0 && h.Offset <= clockLimit) {
				return false
			}
		}
		return true
	}
	if run.Level != nil {
		for _, arena := range run.Level.Rooms {
			if !validArena(arena) {
				return invalid
			}
		}
	}
	if run.Practice != nil && !validArena(run.Practice.Arena) {
		return invalid
	}
	return nil
}
