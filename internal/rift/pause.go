package rift

import "time"

// SetPaused settles completed pause intervals using nondecreasing server timestamps.
// Older paused snapshots without a start timestamp cannot establish a duration.
func (r *Run) SetPaused(paused bool, now time.Time) {
	if r.Status != "fighting" && r.Status != "cleared" {
		return
	}
	stamp := max(r.LastMS, now.UnixMilli())
	if !paused {
		r.recoverConnection(now)
		if r.Paused && r.PauseStartedMS != nil {
			r.Stats.PausedSeconds += float64(max(int64(0), stamp-*r.PauseStartedMS)) / 1000
		}
		r.PauseStartedMS = nil
	} else if !r.Paused || r.PauseStartedMS == nil {
		r.PauseStartedMS = &stamp
	}
	r.Paused = paused
	r.LastMS = stamp
	r.SavedAtMS = stamp
}
