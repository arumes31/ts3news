package rift

import "math"

// CombatStats records confirmed action-mode outcomes, never client estimates.
// It travels with the expedition snapshot and survives seamless room changes.
type CombatStats struct {
	SkillUses      map[string]int     `json:"skill_uses,omitempty"`
	SkillMana      map[string]float64 `json:"skill_mana,omitempty"`
	SkillHits      map[string]int     `json:"skill_hits,omitempty"`
	SkillHealing   map[string]float64 `json:"skill_healing,omitempty"`
	Seconds        float64            `json:"seconds"`
	DamageDealt    float64            `json:"damage_dealt"`
	DamageTaken    float64            `json:"damage_taken"`
	GuardBlocked   float64            `json:"guard_blocked"`
	BarrierBlocked float64            `json:"barrier_blocked"`
	ArmorBlocked   float64            `json:"armor_blocked"`
	Healing        float64            `json:"healing"`
	LargestHit     float64            `json:"largest_hit"`
	ManaSpent      float64            `json:"mana_spent"`
	Kills          int                `json:"kills"`
	Bosses         int                `json:"bosses"`
	Guards         int                `json:"guards"`
	Attacks        int                `json:"attacks"`
	SkillsCast     int                `json:"skills_cast"`
	Jumps          int                `json:"jumps"`
	RoomsCleared   int                `json:"rooms_cleared"`
}

func (r *Run) healPlayer(amount float64) {
	before := r.Player.HP
	r.Player.HP = math.Min(r.Player.MaxHP, r.Player.HP+math.Max(0, amount))
	r.Stats.Healing += math.Max(0, r.Player.HP-before)
}

func (r *Run) healPlayerBySkill(amount float64, skillID string) {
	before := r.Stats.Healing
	r.healPlayer(amount)
	if healed := r.Stats.Healing - before; healed > 0 && skillID != "" {
		if r.Stats.SkillHealing == nil {
			r.Stats.SkillHealing = map[string]float64{}
		}
		r.Stats.SkillHealing[skillID] += healed
	}
}
