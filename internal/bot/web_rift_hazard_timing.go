package bot

import (
	"errors"

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
