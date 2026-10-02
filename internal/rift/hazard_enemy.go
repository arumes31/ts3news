package rift

import "fmt"

// Falling rocks are overhead impacts: airborne enemies are exposed, while
// burrowed enemies and objective props are outside this combat contact rule.
func (r *Run) hurtEnemiesFromHazard(index int, h Hazard) {
	if h.Kind != "falling_rock" || h.Disabled {
		return
	}
	phase := h.Phase(r.Clock)
	if phase < 1.2 || phase >= 1.2+h.Duration {
		return
	}
	region := 0
	if r.Level != nil {
		region = r.Level.Region
	}
	for i := range r.Enemies {
		e := &r.Enemies[i]
		if e.HP <= 0 || e.Burrowed || e.isObjectiveProp() || !contains(h.ContactBounds(r.Clock), e.X, e.Y, 0) {
			continue
		}
		key := fmt.Sprintf("hazard-enemy-%d-%s", index, e.ID)
		shared := "hazard-enemy-hit-" + e.ID
		if r.SkillTimers[key] > 0 || r.SkillTimers[shared] > 0 {
			continue
		}
		r.SkillTimers[key] = h.Period - phase + .05
		r.SkillTimers[shared] = .35
		r.applyEnemyDamage(i, r.hazardContactDamage(region), "hit", 0, true)
	}
}
