package rift

// PendingObjectiveGold is separate from floor drops and ordinary fight gold.
// The per-objective amount is frozen when a mission starts; legacy missions
// without a reward offer do not acquire one retroactively.
func (r *Run) PendingObjectiveGold() int64 {
	o := r.Objectives
	if r.Practice != nil || r.Level == nil || r.Room != len(Rooms)-1 || r.Status != "cleared" || o == nil || !o.Finished || o.Banked || o.Mission != r.Level.ID || o.RewardPerObjective != 5 || o.RewardGold != 0 {
		return 0
	}
	known := map[string]bool{}
	for _, entry := range ObjectiveOptions(r.Build) {
		known[entry.ID] = true
	}
	var gold int64
	for _, entry := range o.Entries {
		if entry.Status == "complete" && known[entry.ID] {
			gold += o.RewardPerObjective
			delete(known, entry.ID)
		}
	}
	return gold
}
