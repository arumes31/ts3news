package bot

import (
	"errors"

	"ts3news/internal/rift"
)

func validateRiftCooldowns(run *rift.Run) error {
	invalid := errors.New("invalid rift snapshot cooldown")
	// Actor attacks and internal grace/hazard timers last only a few seconds.
	// Leave headroom for older content without accepting effectively permanent locks.
	const actorLimit = 60.0
	const authoredLimit = 3600.0
	bounded := func(value, limit float64) bool { return value >= 0 && value <= limit }
	skillLimits := map[string]float64{}
	addSkill := func(skill rift.Skill) bool {
		if !bounded(skill.Cooldown, authoredLimit) {
			return false
		}
		skillLimits[skill.ID] = max(skillLimits[skill.ID], skill.Cooldown)
		return true
	}
	for _, skill := range run.Build.Skills {
		if !addSkill(skill) {
			return invalid
		}
	}
	for _, skill := range run.Build.Signatures {
		if !addSkill(skill) {
			return invalid
		}
	}
	if run.Build.Ultimate != nil && !addSkill(*run.Build.Ultimate) {
		return invalid
	}
	for id, value := range run.SkillTimers {
		if !bounded(value, max(actorLimit, skillLimits[id])) {
			return invalid
		}
	}
	return validateRiftActors(run, func(a rift.Actor) bool { return bounded(a.Cooldown, actorLimit) }, invalid)
}
