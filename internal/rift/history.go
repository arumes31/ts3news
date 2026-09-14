package rift

import "slices"

// MissionHistory contains only confirmed attempts recorded by this version.
// Legacy completion marks are kept separately; they cannot establish timings.
type MissionHistory struct {
	FlawlessTiers    []int          `json:"flawless_tiers,omitempty"`
	FewestHits       *int           `json:"fewest_hits,omitempty"`
	Attempts         int            `json:"attempts"`
	Completions      int            `json:"completions"`
	LastStartedMS    int64          `json:"last_started_ms"`
	LastOutcome      string         `json:"last_outcome"`
	BestSeconds      float64        `json:"best_seconds,omitempty"`
	BestFinishHP     float64        `json:"best_finish_hp,omitempty"`
	BestFinishMaxHP  float64        `json:"best_finish_max_hp,omitempty"`
	CompletedByClass map[string]int `json:"completed_by_class,omitempty"`
}

func (r *Run) beginMissionHistory() {
	if r.Level == nil {
		return
	}
	if r.History == nil {
		r.History = map[int]MissionHistory{}
	}
	h := r.History[r.Level.ID]
	h.Attempts++
	h.LastStartedMS = r.LastMS
	h.LastOutcome = "active"
	r.History[r.Level.ID] = h
	r.MissionStartSeconds = r.Stats.Seconds
	hits := r.Stats.HitsTaken
	r.MissionStartHits = &hits
	r.HistoryActive = true
}

func (r *Run) finishMissionHistory(outcome string) {
	if !r.HistoryActive || r.Level == nil {
		return
	}
	h := r.History[r.Level.ID]
	h.LastOutcome = outcome
	if outcome == "completed" {
		h.Completions++
		if r.MissionStartHits != nil {
			hits := r.Stats.HitsTaken - *r.MissionStartHits
			if hits >= 0 && (h.FewestHits == nil || hits < *h.FewestHits) {
				h.FewestHits = &hits
			}
		}
		if r.Build.Class != "" {
			classes := make(map[string]int, len(h.CompletedByClass)+1)
			for class, count := range h.CompletedByClass {
				classes[class] = count
			}
			classes[r.Build.Class]++
			h.CompletedByClass = classes
		}
		if r.Player.HP > h.BestFinishHP {
			h.BestFinishHP = r.Player.HP
			h.BestFinishMaxHP = r.Player.MaxHP
		}
		elapsed := r.Stats.Seconds - r.MissionStartSeconds
		if elapsed > 0 && (h.BestSeconds == 0 || elapsed < h.BestSeconds) {
			h.BestSeconds = elapsed
		}
	}
	r.History[r.Level.ID] = h
	r.HistoryActive = false
}

// InheritCampaignHistory carries bounded account progress into a newly created
// expedition, whose first attempt has already been recorded by setLevel.
func (r *Run) InheritCampaignHistory(previous *Run) {
	if previous == nil || r.Level == nil {
		return
	}
	current := r.History[r.Level.ID]
	for id, history := range previous.History {
		if id < 1 || id > LevelCount {
			continue
		}
		if history.LastOutcome == "active" {
			history.LastOutcome = "expired"
		}
		r.History[id] = history
	}
	old := r.History[r.Level.ID]
	if _, exists := previous.History[r.Level.ID]; exists {
		current.Attempts += old.Attempts
		current.Completions = old.Completions
		current.BestSeconds = old.BestSeconds
		current.FewestHits = old.FewestHits
		current.BestFinishHP = old.BestFinishHP
		current.BestFinishMaxHP = old.BestFinishMaxHP
		current.CompletedByClass = old.CompletedByClass
		current.FlawlessTiers = old.FlawlessTiers
	}
	r.History[r.Level.ID] = current
	r.CompletedLevels = append([]int(nil), previous.CompletedLevels...)
}

func (r *Run) recordFlawlessRoom() {
	if !r.HistoryActive || r.Level == nil || r.RoomStartHits == nil || r.Stats.HitsTaken != *r.RoomStartHits {
		return
	}
	h := r.History[r.Level.ID]
	tier := r.Room + 1
	if slices.Contains(h.FlawlessTiers, tier) {
		return
	}
	h.FlawlessTiers = append(append([]int(nil), h.FlawlessTiers...), tier)
	slices.Sort(h.FlawlessTiers)
	r.History[r.Level.ID] = h
}
