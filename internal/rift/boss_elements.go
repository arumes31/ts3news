package rift

import "ts3news/internal/content"

// BossElementPhase freezes the ward and its canonical weakness for one phase.
// Absence on a saved boss preserves that encounter's original damage rules.
type BossElementPhase struct {
	Element  content.Element `json:"element"`
	Weakness content.Element `json:"weakness"`
}

func bossElementPhases(base content.Element) [3]BossElementPhase {
	cycle := []content.Element{content.ElementFire, content.ElementAir, content.ElementEarth, content.ElementWater}
	for index, element := range cycle {
		if element != base {
			continue
		}
		var phases [3]BossElementPhase
		for phase := range phases {
			defense := cycle[(index+phase)%len(cycle)]
			phases[phase] = BossElementPhase{Element: defense, Weakness: content.ElementWeakness(defense)}
		}
		return phases
	}
	return [3]BossElementPhase{}
}

func (a *Actor) elementalPhase() *BossElementPhase {
	if a.Kind != "boss" || a.ElementalPhases == [3]BossElementPhase{} {
		return nil
	}
	return &a.ElementalPhases[max(0, min(2, a.Phase-1))]
}

// Only direct attacks call this helper. Environmental and legacy effect-only
// damage retain their existing rules; artwork is never an element source.
func (r *Run) hurtEnemyElement(index int, damage float64, effect string, pierce float64, element content.Element) {
	if ward := r.Enemies[index].elementalPhase(); ward != nil {
		damage *= content.ElementMultiplier(element, ward.Element)
	}
	r.hurtEnemyPiercing(index, damage, effect, pierce)
}
