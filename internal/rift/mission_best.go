package rift

import "maps"

// MissionBest retains comparable metrics independently of lifetime completion counts.
type MissionBest struct {
	BestSeconds      float64 `json:"best_seconds,omitempty"`
	BestSecondsAtMS  int64   `json:"best_seconds_at_ms,omitempty"`
	BestFinishHP     float64 `json:"best_finish_hp,omitempty"`
	BestFinishMaxHP  float64 `json:"best_finish_max_hp,omitempty"`
	BestFinishHPAtMS int64   `json:"best_finish_hp_at_ms,omitempty"`
	FewestHits       *int    `json:"fewest_hits,omitempty"`
	FewestHitsAtMS   int64   `json:"fewest_hits_at_ms,omitempty"`
	FlawlessTiers    []int   `json:"flawless_tiers,omitempty"`
}

func (h MissionHistory) best() MissionBest {
	return MissionBest{BestSeconds: h.BestSeconds, BestSecondsAtMS: h.BestSecondsAtMS, BestFinishHP: h.BestFinishHP, BestFinishMaxHP: h.BestFinishMaxHP, BestFinishHPAtMS: h.BestFinishHPAtMS, FewestHits: h.FewestHits, FewestHitsAtMS: h.FewestHitsAtMS, FlawlessTiers: h.FlawlessTiers}
}

func (h *MissionHistory) selectDefinition(definition string) {
	if h.Definition == definition {
		return
	}
	versions := maps.Clone(h.Versions)
	if versions == nil {
		versions = map[string]MissionBest{}
	}
	if h.BestSeconds > 0 || h.BestFinishHP > 0 || h.FewestHits != nil || len(h.FlawlessTiers) > 0 {
		versions[h.Definition] = h.best()
	}
	best := versions[definition]
	h.BestSeconds = best.BestSeconds
	h.BestSecondsAtMS = best.BestSecondsAtMS
	h.BestFinishHP = best.BestFinishHP
	h.BestFinishMaxHP = best.BestFinishMaxHP
	h.BestFinishHPAtMS = best.BestFinishHPAtMS
	h.FewestHits = best.FewestHits
	h.FewestHitsAtMS = best.FewestHitsAtMS
	h.FlawlessTiers = best.FlawlessTiers
	h.Definition = definition
	h.Versions = versions
}
