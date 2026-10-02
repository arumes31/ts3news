package rift

// bankObjectives is called only after checkpoint rewards are banked successfully.
func (r *Run) bankObjectives() {
	o := r.Objectives
	if r.Practice != nil || r.Level == nil || r.Room != len(Rooms)-1 || r.Status != "cleared" || o == nil || !o.Finished || o.Banked || o.Mission != r.Level.ID {
		return
	}
	difficulty := o.Difficulty
	if difficulty == "" {
		difficulty = r.Level.Difficulty
	}
	if difficulty == "" {
		return
	}
	if r.ObjectiveHistory == nil {
		r.ObjectiveHistory = map[string]map[string]int{}
	}
	if r.ObjectiveHistory[difficulty] == nil {
		r.ObjectiveHistory[difficulty] = map[string]int{}
	}
	for _, entry := range o.Entries {
		if entry.Status == "complete" {
			r.ObjectiveHistory[difficulty][entry.ID]++
		}
	}
	o.Difficulty = difficulty
	o.Banked = true
}

func (r *Run) inheritObjectiveHistory(previous *Run) {
	combined := map[string]map[string]int{}
	for _, history := range []map[string]map[string]int{previous.ObjectiveHistory, r.ObjectiveHistory} {
		for difficulty, entries := range history {
			if combined[difficulty] == nil {
				combined[difficulty] = map[string]int{}
			}
			for id, count := range entries {
				combined[difficulty][id] += count
			}
		}
	}
	r.ObjectiveHistory = combined
}
