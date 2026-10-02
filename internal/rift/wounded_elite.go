package rift

// Elite priorities latch once per encounter, so healing cannot repeatedly reset
// warnings or replay the phase cue. Existing attack and recovery timers survive.
func (r *Run) updateElitePriorities(e *Actor) {
	if !e.Elite || e.Enraged || !(e.MaxHP > 0 && e.HP > 0 && e.HP <= e.MaxHP*.35) {
		return
	}
	if e.Kind != "knight" && e.Kind != "goblin" && e.Kind != "archer" {
		return
	}
	e.Enraged = true
	if e.Kind != "archer" {
		e.Charging = true
	}
	r.eventAtHeight("elite_desperation", e.X, e.Y-25, 0, e.Elevation)
}

func archerRetreatDistance(e *Actor) float64 {
	if e.Elite && e.Enraged {
		return 220
	}
	return 150
}
