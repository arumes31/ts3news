package rift

// AttemptRecord retains a bounded recent outcome without changing personal bests.
type AttemptRecord struct {
	Mission int     `json:"mission"`
	Outcome string  `json:"outcome"`
	AtMS    int64   `json:"at_ms"`
	Class   string  `json:"class"`
	Seconds float64 `json:"seconds"`
	HP      float64 `json:"hp"`
	MaxHP   float64 `json:"max_hp"`
	Hits    *int    `json:"hits,omitempty"`
}

func (r *Run) attemptSnapshot(outcome string, at int64) AttemptRecord {
	record := AttemptRecord{Mission: r.Level.ID, Outcome: outcome, AtMS: at, Class: r.Build.Class, Seconds: max(0, r.Stats.Seconds-r.MissionStartSeconds), HP: r.Player.HP, MaxHP: r.Player.MaxHP}
	if r.MissionStartHits != nil {
		hits := r.Stats.HitsTaken - *r.MissionStartHits
		if hits >= 0 {
			record.Hits = &hits
		}
	}
	return record
}

func (r *Run) appendAttempt(record AttemptRecord) {
	r.AttemptHistory = append(r.AttemptHistory, record)
	if len(r.AttemptHistory) > 50 {
		r.AttemptHistory = append([]AttemptRecord(nil), r.AttemptHistory[len(r.AttemptHistory)-50:]...)
	}
}
