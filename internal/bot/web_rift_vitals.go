package bot

import (
	"errors"
	"math"

	"ts3news/internal/rift"
)

// validateRiftVitals checks actors that can enter simulation, including future waves.
// Reject corrupt saves instead of silently changing combat state or earned rewards.
func validateRiftVitals(run *rift.Run) error {
	valid := func(a rift.Actor) bool {
		return a.MaxHP > 0 && !math.IsInf(a.MaxHP, 0) && a.HP >= 0 && a.HP <= a.MaxHP && a.Mana >= 0 && a.Mana <= 100
	}
	return validateRiftActors(run, valid, errors.New("invalid rift snapshot health or mana"))
}

// validateRiftActors includes templates that can reenter simulation after a reset.
func validateRiftActors(run *rift.Run, valid func(rift.Actor) bool, invalid error) error {
	if !valid(run.Player) {
		return invalid
	}
	if run.Practice != nil && run.Practice.BossStart != nil && !valid(*run.Practice.BossStart) {
		return invalid
	}
	for _, a := range run.Enemies {
		if !valid(a) {
			return invalid
		}
	}
	for _, room := range run.EncounterPlan {
		for _, a := range room {
			if !valid(a) {
				return invalid
			}
		}
	}
	if o := run.RoomObjective; o != nil {
		if o.Lantern != nil && !valid(*o.Lantern) {
			return invalid
		}
		if o.Escort != nil && !valid(*o.Escort) {
			return invalid
		}
		for _, lane := range o.Lanes {
			if !valid(lane.Ward) {
				return invalid
			}
		}
		for _, wave := range o.Waves {
			for _, a := range wave {
				if !valid(a) {
					return invalid
				}
			}
		}
	}
	return nil
}
