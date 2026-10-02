package rift

import (
	"errors"
	"math"
)

// UseHealingPotion applies an already-authorized inventory use. The caller must
// consume one owned potion and persist this run in the same transaction.
func (r *Run) UseHealingPotion(amount float64) error {
	if r.Practice != nil || r.Status != "fighting" || r.Paused || r.Player.HP <= 0 || r.Player.HP >= r.Player.MaxHP {
		return errors.New("healing potion unavailable now")
	}
	if amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return errors.New("invalid healing potion amount")
	}
	if r.SkillTimers["healing_potion"] > 0 {
		return errors.New("healing potion is cooling down")
	}
	if r.SkillTimers == nil {
		r.SkillTimers = map[string]float64{}
	}
	r.healPlayer(amount)
	r.Stats.PotionsUsed++
	r.SkillTimers["healing_potion"] = 8
	r.UpdateObjectives()
	return nil
}
