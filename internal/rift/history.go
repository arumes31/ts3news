package rift

import "slices"

// ClearResult preserves the most recent completed mission result through advancement.
type ClearResult struct {
	Mission int      `json:"mission"`
	First   bool     `json:"first"`
	Records []string `json:"records"`
}

// MissionHistory contains only confirmed attempts recorded by this version.
// Legacy completion marks are kept separately; they cannot establish timings.
type MissionHistory struct {
	Definition string `json:"definition,omitempty"`
	Versions map[string]MissionBest `json:"versions,omitempty"`
	BestSecondsAtMS  int64          `json:"best_seconds_at_ms,omitempty"`
	BestFinishHPAtMS int64          `json:"best_finish_hp_at_ms,omitempty"`
	FewestHitsAtMS   int64          `json:"fewest_hits_at_ms,omitempty"`
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
	r.MissionDefinition = levelDefinition(r.Level)
	h.selectDefinition(r.MissionDefinition)
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
	comparable := r.MissionDefinition != "" && h.Definition == r.MissionDefinition
	if outcome == "completed" {
		r.LastClear = &ClearResult{Mission: r.Level.ID, First: h.Completions == 0 && !slices.Contains(r.CompletedLevels, r.Level.ID), Records: []string{}}
		r.ClearStreak++
		r.BestClearStreak = max(r.BestClearStreak, r.ClearStreak)
		h.Completions++
		if comparable && r.MissionStartHits != nil {
			hits := r.Stats.HitsTaken - *r.MissionStartHits
			if hits >= 0 && (h.FewestHits == nil || hits < *h.FewestHits) {
				h.FewestHits = &hits
				h.FewestHitsAtMS = r.LastMS
				r.LastClear.Records = append(r.LastClear.Records, "hits")
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
		if comparable && r.Player.HP > h.BestFinishHP {
			r.LastClear.Records = append(r.LastClear.Records, "health")
			h.BestFinishHP = r.Player.HP
			h.BestFinishHPAtMS = r.LastMS
			h.BestFinishMaxHP = r.Player.MaxHP
		}
		elapsed := r.Stats.Seconds - r.MissionStartSeconds
		r.recordRegionTime(elapsed)
		if comparable && elapsed > 0 && (h.BestSeconds == 0 || elapsed < h.BestSeconds) {
			h.BestSeconds = elapsed
			h.BestSecondsAtMS = r.LastMS
			r.LastClear.Records = append(r.LastClear.Records, "time")
		}
	} else {
		r.RegionAttempt = nil
		r.ClearStreak = 0
	}
	r.History[r.Level.ID] = h
	r.appendAttempt(r.attemptSnapshot(outcome, r.LastMS))
	r.HistoryActive = false
}

// InheritCampaignHistory carries bounded account progress into a newly created
// expedition, whose first attempt has already been recorded by setLevel.
func (r *Run) InheritCampaignHistory(previous *Run) {
	if previous == nil || r.Level == nil || r.Practice != nil || previous.Practice != nil {
		return
	}
	r.LastObjectives = previous.LastObjectives
	if previous.Objectives != nil && previous.Objectives.Finished {
		r.LastObjectives = previous.Objectives
	}
	r.inheritRegionRecords(previous)
	r.inheritMonsterRecords(previous)
	r.inheritObjectiveHistory(previous)
	r.PastExpeditions = previous.RecordedTotals()
	r.AttemptHistory = append([]AttemptRecord(nil), previous.AttemptHistory...)
	if previous.LastEncounter != nil {
		r.LastEncounter = previous.LastEncounter
	}
	if previous.HistoryActive && previous.Level != nil {
		r.appendAttempt(previous.attemptSnapshot("expired", r.LastMS))
	}
	current := r.History[r.Level.ID]
	r.BestClearStreak = previous.BestClearStreak
	if previous.Status == "complete" || previous.Status == "banked" {
		r.ClearStreak = previous.ClearStreak
	}
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
        old.selectDefinition(r.MissionDefinition)
        old.Attempts += current.Attempts
        old.LastStartedMS = current.LastStartedMS
        old.LastOutcome = current.LastOutcome
        current = old
	}
	r.History[r.Level.ID] = current
	r.CompletedLevels = append([]int(nil), previous.CompletedLevels...)
}

func (r *Run) recordFlawlessRoom() {
	if !r.HistoryActive || r.Level == nil || r.RoomStartHits == nil || r.Stats.HitsTaken != *r.RoomStartHits {
		return
	}
	h := r.History[r.Level.ID]
	if r.MissionDefinition == "" || h.Definition != r.MissionDefinition {
		return
	}
	tier := r.Room + 1
	if slices.Contains(h.FlawlessTiers, tier) {
		return
	}
	h.FlawlessTiers = append(append([]int(nil), h.FlawlessTiers...), tier)
	slices.Sort(h.FlawlessTiers)
	r.History[r.Level.ID] = h
}

// CareerTotals combines recorded expeditions; missing older history is not inferred.
type CareerTotals struct {
	PerfectGuards int `json:"perfect_guards"`
	Enemies         int   `json:"enemies"`
	Bosses          int   `json:"bosses"`
	TreasureGoblins int   `json:"treasure_goblins"`
	Gold            int64 `json:"gold"`
	Gear            int   `json:"gear"`
}

// RecordedTotals adds the current expedition to its immutable earlier totals.
// Only confirmed banking contributes rewards, even if the expedition is lost.
func (r *Run) RecordedTotals() CareerTotals {
	totals := r.PastExpeditions
	totals.PerfectGuards += r.Stats.PerfectGuards
	totals.Enemies += r.Stats.Kills
	totals.Bosses += r.Stats.Bosses
	totals.TreasureGoblins += r.Stats.TreasureGoblins
	totals.Gold += r.BankedGold
	totals.Gear += r.TotalBankedItems()
	return totals
}
