package bot

import (
	"errors"

	"ts3news/internal/rift"
)

func validateRiftPositions(run *rift.Run) error {
	invalid := errors.New("invalid rift snapshot actor position")
	// Match the simulation's player movement rectangle. Objective actors can sit
	// outside that lane, so their coordinates use the full rendered arena bounds.
	p := run.Player
	if !(p.X >= 35 && p.X <= rift.Width-35 && p.Y >= 315 && p.Y <= 490) {
		return invalid
	}
	x := func(value float64) bool { return value >= 0 && value <= rift.Width }
	y := func(value float64) bool { return value >= 0 && value <= 540 }
	return validateRiftActors(run, func(a rift.Actor) bool {
		// Zero is also the legacy/unset sentinel for targets and navigation hints.
		return x(a.X) && y(a.Y) && x(a.TargetX) && y(a.TargetY) && x(a.RouteX) && y(a.RouteY) && x(a.LedgeRouteX) && x(a.PatrolOriginX)
	}, invalid)
}
