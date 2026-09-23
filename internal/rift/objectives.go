package rift

// ObjectiveProgress is a server-confirmed optional mission challenge.
type ObjectiveProgress struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Current     float64 `json:"current"`
	Target      float64 `json:"target"`
	Status      string  `json:"status"`
	Reason      string  `json:"reason,omitempty"`
}

type MissionObjectives struct {
	Banked       bool                `json:"banked,omitempty"`
	Difficulty   string              `json:"difficulty,omitempty"`
	Mission      int                 `json:"mission"`
	Finished     bool                `json:"finished"`
	StartSeconds float64             `json:"start_seconds"`
	StartDamage  float64             `json:"start_damage"`
	StartSkills  int                 `json:"start_skills"`
	StartGuards  int                 `json:"start_guards"`
	Entries      []ObjectiveProgress `json:"entries"`
}

// ObjectiveOptions supplies both mission tracking and the pre-run preview.
func ObjectiveOptions() []ObjectiveProgress {
	return []ObjectiveProgress{
		{ID: "timed", Name: "Swift clear", Description: "Clear all three tiers within 180 combat seconds. Pauses do not count.", Target: 180, Status: "active"},
		{ID: "no_damage", Name: "Untouched", Description: "Clear all three tiers without taking health damage.", Status: "active"},
		{ID: "basic_only", Name: "Basic attacks only", Description: "Clear all three tiers without using abilities. Movement, jumping and guarding are allowed.", Status: "active"},
		{ID: "guard", Name: "Guard mastery", Description: "Block at least five attacks and clear all three tiers.", Target: 5, Status: "active"},
	}
}

func (r *Run) beginObjectives() {
	if r.Objectives != nil && r.Objectives.Finished {
		r.LastObjectives = r.Objectives
	}
	r.Objectives = &MissionObjectives{Mission: r.Level.ID, Difficulty: r.Level.Difficulty, StartSeconds: r.Stats.Seconds, StartDamage: r.Stats.DamageTaken, StartSkills: r.Stats.SkillsCast, StartGuards: r.Stats.Guards, Entries: ObjectiveOptions()}
}

// UpdateObjectives refreshes progress from confirmed combat and finalizes ended runs.
func (r *Run) UpdateObjectives() {
	o := r.Objectives
	if o == nil || o.Finished || r.Practice != nil {
		return
	}
	cleared := r.Room == len(Rooms)-1 && (r.Status == "cleared" || r.Status == "complete" || r.Status == "banked")
	ended := cleared || r.Status == "defeated" || r.Status == "banked" || r.Status == "expired"
	for i := range o.Entries {
		e := &o.Entries[i]
		if e.Status == "failed" {
			continue
		}
		reason := ""
		switch e.ID {
		case "timed":
			e.Current = max(0, r.Stats.Seconds-o.StartSeconds)
			if e.Current > e.Target {
				reason = "Exceeded 180 combat seconds."
			}
		case "no_damage":
			e.Current = max(0, r.Stats.DamageTaken-o.StartDamage)
			if e.Current > 0 {
				reason = "Took health damage."
			}
		case "basic_only":
			e.Current = float64(max(0, r.Stats.SkillsCast-o.StartSkills))
			if e.Current > 0 {
				reason = "Used an ability."
			}
		case "guard":
			e.Current = float64(max(0, r.Stats.Guards-o.StartGuards))
			if cleared && e.Current < e.Target {
				reason = "Finished with fewer than five blocks."
			}
		}
		if reason != "" {
			e.Status = "failed"
			e.Reason = reason
		} else if cleared {
			e.Status = "complete"
		} else if ended {
			e.Status = "failed"
			e.Reason = "Mission ended before all three tiers were cleared."
		}
	}
	o.Finished = ended
}
