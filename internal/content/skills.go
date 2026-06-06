package content

import (
	"fmt"
	"math/rand"
	"strings"
)

type SkillType string

const (
	SkillPhysical SkillType = "Physical"
	SkillMagic    SkillType = "Magic"
	SkillBuff     SkillType = "Buff"
	SkillDebuff   SkillType = "Debuff"
)

type Skill struct {
	ID          string
	Name        string
	Type        SkillType
	Rarity      Rarity
	Power       float64 // Multiplier for damage/effect
	IgnoreDef   float64 // Percentage (0.0 - 1.0)
	StunChance  float64 // Percentage (0.0 - 1.0)
	HealPercent float64 // Percentage of max HP
	Description string
	Special     ItemEffect
}

var allSkills []Skill

func init() {
	// Inspired Prefix & Action pools for 1500+ variants
	prefixes := []string{
		"Mortal", "Heroic", "Flash", "Greater", "Lesser", "Chaos", "Fel", "Shadow", "Holy", "Frost",
		"Fire", "Arcane", "Divine", "Primal", "Ancient", "Abyssal", "Spectral", "Vengeful", "Spiteful", "Cursed",
		"Hallowed", "Glacial", "Volcanic", "Static", "Thunderous", "Corrupting", "Blighted", "Toxic", "Metallic", "Glass",
		"Lunar", "Solar", "Celestial", "Infernal", "Mystic", "Raging", "Silent", "Eternal", "Void", "Astral",
		"Iron", "Steel", "Mithril", "Adamant", "Crystalline", "Nebulous", "Star-Forged", "Storm-Born", "Shadow-Bound", "Light-Blessed",
	}
	actions := []string{
		"Strike", "Blast", "Roar", "Slash", "Burst", "Touch", "Winds", "Nova", "Pulse", "Drain",
		"Bolt", "Ray", "Wave", "Aura", "Shield", "Plea", "Call", "Fury", "Vortex", "Sunder",
		"Mend", "Heal", "Bash", "Cleave", "Execute", "Rend", "Charge", "Leap", "Smite", "Shock",
		"Breath", "Bite", "Sting", "Claw", "Maul", "Swipe", "Growl", "Prowl", "Shred", "Blink",
	}

	idx := 1
	for _, p := range prefixes {
		for _, a := range actions {
			// Generate 5 rarity tiers for every name combination (50 * 40 * 5 = 10,000 potential skills)
			// But let's keep it manageable at ~1500 by using a selection logic
			for rIdx := 0; rIdx < 5; rIdx++ {
				rarity := Rarity(rIdx)
				name := p + " " + a
				
				// Only keep ~30% of combinations to reach ~1500-2000 total
				if (idx+rIdx)%3 != 0 {
					continue
				}

				sType := SkillPhysical
				if strings.Contains(name, "Bolt") || strings.Contains(name, "Blast") || strings.Contains(name, "Nova") { sType = SkillMagic }
				if strings.Contains(name, "Heal") || strings.Contains(name, "Mend") || strings.Contains(name, "Shield") { sType = SkillBuff }
				if strings.Contains(name, "Curse") || strings.Contains(name, "Sunder") || strings.Contains(name, "Drain") { sType = SkillDebuff }

				s := Skill{
					ID:     fmt.Sprintf("S%d_%d", idx, rIdx),
					Name:   name,
					Type:   sType,
					Rarity: rarity,
					Power:  1.2 + (0.6 * float64(rarity)),
				}

				// Mechanics
				if strings.Contains(name, "Sunder") || strings.Contains(name, "Execute") { s.IgnoreDef = 0.3 + (0.1 * float64(rarity)) }
				if strings.Contains(name, "Bash") || strings.Contains(name, "Shock") { s.StunChance = 0.1 + (0.05 * float64(rarity)) }
				if strings.Contains(name, "Heal") || strings.Contains(name, "Mend") { s.HealPercent = 0.1 + (0.05 * float64(rarity)) }
				
				// Rare special effects
				if rarity >= RarityEpic && rand.Float64() < 0.1 {
					s.Special = EffectMindControl
				}
				if rarity == RarityLegendary && rand.Float64() < 0.05 {
					s.Special = EffectPhoenix
				}

				s.Description = fmt.Sprintf("%s %s rank %d.", s.Rarity, s.Name, int(rarity)+1)
				allSkills = append(allSkills, s)
			}
			idx++
		}
	}
}

func (s Skill) Score() int {
	score := int(s.Power*100) + int(s.IgnoreDef*100) + int(s.StunChance*100) + int(s.HealPercent*100)
	if s.Special == EffectMindControl { score += 500 }
	if s.Special == EffectPhoenix { score += 1000 }
	return score
}

func RandomSkill() Skill {
	s := allSkills[rand.Intn(len(allSkills))]
	// Roll for additional effect if it doesn't have one
	if s.Special == EffectNone {
		s.Special = RandomItemEffect()
	}
	return s
}

func GetSkillByID(id string) (Skill, bool) {
	for _, s := range allSkills {
		if s.ID == id {
			return s, true
		}
	}
	return Skill{}, false
}

func IsSkill(name string) bool {
	for _, s := range allSkills {
		if s.Name == name {
			return true
		}
	}
	return false
}
