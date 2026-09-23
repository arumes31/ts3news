package rift

import "ts3news/internal/content"

func isPriorityElite(enemy Actor) bool {
	if enemy.Kind == "boss" {
		return false
	}
	switch content.MobType(enemy.Tier) {
	case content.MobEliteMinion, content.MobElite, content.MobMiniboss:
		return true
	}
	return enemy.Tier == "" && enemy.Kind == "knight"
}

// recordPriorityDefeat checks the living lineup at the confirmed defeat, before
// later damage in the same tick can remove a priority target.
func (r *Run) recordPriorityDefeat(enemy Actor) {
	if r.Practice != nil || r.Objectives == nil || r.Objectives.Finished {
		return
	}
	rangedAlive, eliteAlive := false, false
	for _, other := range r.Enemies {
		if other.HP > 0 {
			rangedAlive = rangedAlive || other.Kind == "archer"
			eliteAlive = eliteAlive || isPriorityElite(other)
		}
	}
	for i := range r.Objectives.Entries {
		entry := &r.Objectives.Entries[i]
		if entry.Status != "active" {
			continue
		}
		reason := ""
		if entry.ID == "ranged_priority" && enemy.Kind != "archer" && rangedAlive {
			reason = "Defeated another enemy while a ranged enemy was still alive in this tier."
		}
		if entry.ID == "elite_priority" && !isPriorityElite(enemy) && eliteAlive {
			reason = "Defeated another enemy while an elite was still alive in this tier."
		}
		if reason != "" {
			entry.Current++
			entry.Status = "failed"
			entry.Reason = reason
		}
	}
}
