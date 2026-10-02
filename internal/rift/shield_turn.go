package rift

import "math"

const shieldGuardTurnSeconds = .35

// guardShieldTurn braces nearby shield carriers between attacks. Crossing their
// facing direction opens a short rear-strike window; leaving guard cancels it.
func guardShieldTurn(e *Actor, dx, dy, dt float64) bool {
	if (e.Kind != "knight" && !e.Shield) || e.Cooldown <= 0 || e.Windup > 0 || e.PoseTime > 0 || math.Abs(dx) > 120 || math.Abs(dy) > 50 {
		e.TurnDelay = 0
		if dx != 0 {
			e.Facing = math.Copysign(1, dx)
		}
		return false
	}
	e.Guard, e.Pose = true, "guard"
	if dx == 0 || e.Facing == math.Copysign(1, dx) {
		e.TurnDelay = 0
		return true
	}
	if e.Facing == 0 {
		e.Facing = math.Copysign(1, dx)
		e.TurnDelay = 0
		return true
	}
	if e.TurnDelay <= 0 || !finiteShieldTurn(e.TurnDelay) {
		e.TurnDelay = shieldGuardTurnSeconds
	}
	e.TurnDelay = math.Max(0, math.Min(shieldGuardTurnSeconds, e.TurnDelay)-dt)
	if e.TurnDelay <= 1e-9 {
		e.Facing = math.Copysign(1, dx)
		e.TurnDelay = 0
	}
	return true
}

func finiteShieldTurn(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
