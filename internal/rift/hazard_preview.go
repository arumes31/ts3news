package rift

import "math"

// HazardContactHealthLoss predicts one accepted arena-hazard contact using the
// current defenses. Evasion, contact cooldowns and temporary immunity are excluded:
// callers must describe this as damage IF hit, not an inevitable incoming hit.
// Keep parity with hurtPlayerGuardable; the preview never consumes defenses.
func (r *Run) HazardContactHealthLoss() float64 {
	if r.Player.HP <= 0 {
		return 0
	}
	region := 0
	if r.Level != nil {
		region = r.Level.Region
	}
	damage := r.hazardContactDamage(region)
	if r.FirstHitGrace {
		damage *= .5
	}
	damage = math.Max(2, damage-r.Build.Armor*.4)
	// Floor contact originates at the player's position, so guard is frontal.
	if r.Player.Guard {
		damage *= .18
	}
	damage = math.Max(0, damage-r.Barrier)
	return math.Min(r.Player.HP, damage)
}
