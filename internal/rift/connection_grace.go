package rift

import "time"

const (
	// connectionGraceSeconds is how long recovery suppresses incoming damage.
	connectionGraceSeconds = 1.2
	// connectionGraceRearmSeconds is the simulated time after a recovery before
	// another one may start. It covers the protected interval plus an
	// unprotected interval, so request timing cannot chain recoveries into
	// permanent damage immunity while offensive progress continues.
	connectionGraceRearmSeconds = connectionGraceSeconds + 3.0
)

// recoverConnection gives an active fighter time to react after missing updates.
// Explicit pauses are excluded; their clock is already safely frozen.
func (r *Run) recoverConnection(now time.Time) {
	if r.Paused || r.Status != "fighting" || r.Player.HP <= 0 || now.UnixMilli()-r.LastMS < 2000 {
		return
	}
	if r.SkillTimers == nil {
		r.SkillTimers = map[string]float64{}
	}
	if r.SkillTimers["connection_grace"] > 0 || r.SkillTimers["connection_grace_rearm"] > 0 {
		return
	}
	r.SkillTimers["connection_grace"] = connectionGraceSeconds
	r.SkillTimers["connection_grace_rearm"] = connectionGraceRearmSeconds
	r.FirstHitGrace = true
}
