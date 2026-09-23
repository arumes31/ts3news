package rift

// recordPriorityDefeat checks the living lineup at the confirmed defeat, before
// later damage in the same tick can remove a priority target.
func (r *Run) recordPriorityDefeat(enemy Actor) {
	if r.Practice != nil || r.Objectives == nil || r.Objectives.Finished || enemy.Kind == "archer" {
		return
	}
	rangedAlive := false
	for _, other := range r.Enemies {
		rangedAlive = rangedAlive || other.Kind == "archer" && other.HP > 0
	}
	if !rangedAlive {
		return
	}
	for i := range r.Objectives.Entries {
		entry := &r.Objectives.Entries[i]
		if entry.ID == "ranged_priority" && entry.Status == "active" {
			entry.Current++
			entry.Status = "failed"
			entry.Reason = "Defeated another enemy while a ranged enemy was still alive in this tier."
		}
	}
}
