package rift

import "math"

// CombatStats records confirmed action-mode outcomes, never client estimates.
// It travels with the expedition snapshot and survives seamless room changes.
type CombatStats struct {
	Dodges           int                `json:"dodges"`
	NonMeleeCasts    int                `json:"non_melee_casts"`
	AerialFinishes   int                `json:"aerial_finishes"`
	HazardContacts   int                `json:"hazard_contacts"`
	UltimateCasts    int                `json:"ultimate_casts"`
	PausedSeconds    float64            `json:"paused_seconds"`
	TreasureGoblins  int                `json:"treasure_goblins"`
	HighestCombo     int                `json:"highest_combo"`
	HitsTaken        int                `json:"hits_taken"`
	SkillUses        map[string]int     `json:"skill_uses,omitempty"`
	SkillMana        map[string]float64 `json:"skill_mana,omitempty"`
	SkillHits        map[string]int     `json:"skill_hits,omitempty"`
	SkillHealing     map[string]float64 `json:"skill_healing,omitempty"`
	SkillBarrier     map[string]float64 `json:"skill_barrier,omitempty"`
	ChargedFinishers int                `json:"charged_finishers"`
	EmptyFinishers   int                `json:"empty_finishers"`
	ChargesSpent     int                `json:"charges_spent"`
	Seconds          float64            `json:"seconds"`
	DamageDealt      float64            `json:"damage_dealt"`
	DamageTaken      float64            `json:"damage_taken"`
	GuardBlocked     float64            `json:"guard_blocked"`
	BarrierBlocked   float64            `json:"barrier_blocked"`
	ArmorBlocked     float64            `json:"armor_blocked"`
	Healing          float64            `json:"healing"`
	LargestHit       float64            `json:"largest_hit"`
	ManaSpent        float64            `json:"mana_spent"`
	Kills            int                `json:"kills"`
	Bosses           int                `json:"bosses"`
	Guards           int                `json:"guards"`
	Attacks          int                `json:"attacks"`
	SkillsCast       int                `json:"skills_cast"`
	Jumps            int                `json:"jumps"`
	RoomsCleared     int                `json:"rooms_cleared"`
}

func (r *Run) healPlayer(amount float64) {
	before := r.Player.HP
	r.Player.HP = math.Min(r.Player.MaxHP, r.Player.HP+math.Max(0, amount))
	healed := math.Max(0, r.Player.HP-before)
	r.Stats.Healing += healed
	if healed > 0 {
		r.eventAtHeight("heal", r.Player.X, r.Player.Y-35, healed, r.Player.Elevation)
	}
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

func (r *Run) addBarrier(amount float64, skillID string) {
	before := r.Barrier
	r.Barrier = math.Min(r.Player.MaxHP*.5, r.Barrier+math.Max(0, amount))
	added := math.Max(0, r.Barrier-before)
	if added > 0 {
		if skillID != "" {
			if r.BarrierSources == nil {
				r.BarrierSources = map[string]float64{}
			}
			r.BarrierSources[skillID] += added
		}
		r.eventAtHeight("barrier", r.Player.X, r.Player.Y-35, added, r.Player.Elevation)
	}
}

// Mixed barriers share absorption in proportion to their remaining strength.
// Any shield from an older snapshot without sources stays unattributed.
func (r *Run) absorbBarrier(damage float64) float64 {
	if r.Barrier <= 0 {
		return 0
	}
	absorbed := math.Min(math.Max(0, damage), r.Barrier)
	for skillID, remaining := range r.BarrierSources {
		share := math.Min(remaining, absorbed*remaining/r.Barrier)
		if share > 0 {
			if r.Stats.SkillBarrier == nil {
				r.Stats.SkillBarrier = map[string]float64{}
			}
			r.Stats.SkillBarrier[skillID] += share
		}
		if absorbed == r.Barrier || remaining <= share {
			delete(r.BarrierSources, skillID)
		} else {
			r.BarrierSources[skillID] = remaining - share
		}
	}
	r.Barrier -= absorbed
	r.Stats.BarrierBlocked += absorbed
	return absorbed
}

// recordDodge records an enemy attack that actually intersected an airborne player.
func (r *Run) recordDodge() {
	r.Stats.Dodges++
	r.event("dodge", r.Player.X, r.Player.Y, 0)
}
