package rift

import "time"

// recoverConnection gives an active fighter time to react after missing updates.
// Explicit pauses are excluded; their clock is already safely frozen.
func (r *Run) recoverConnection(now time.Time) {
	if r.Paused || r.Status != "fighting" || r.Player.HP <= 0 || now.UnixMilli()-r.LastMS < 2000 {
		return
	}
	if r.SkillTimers == nil {
		r.SkillTimers = map[string]float64{}
	}
	r.SkillTimers["connection_grace"] = 1.2
	r.FirstHitGrace = true
}
