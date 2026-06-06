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
}

var allSkills []Skill

func init() {
	// Procedurally generate 300+ skills
	prefixes := []string{
		"Fiery", "Frosty", "Thunderous", "Corrupting", "Divine", "Abyssal", "Spectral", "Ancient", "Primal", "Cursed",
		"Gilded", "Shattered", "Void", "Hallowed", "Toxic", "Metallic", "Stormy", "Shadowy", "Luminous", "Blighted",
		"Arcane", "Raging", "Silent", "Eternal", "Volcanic", "Glacial", "Static", "Celestial", "Infernal", "Mystic",
	}
	actions := []string{
		"Strike", "Blast", "Roar", "Slash", "Burst", "Touch", "Winds", "Nova", "Pulse", "Drain",
		"Bolt", "Ray", "Wave", "Aura", "Shield", "Plea", "Call", "Fury", "Vortex", "Sunder",
	}

	idx := 1
	for _, p := range prefixes {
		for _, a := range actions {
			name := p + " " + a
			rarity := Rarity(idx % 5)
			
			// Balance based on action type or index
			sType := SkillPhysical
			if idx%2 == 0 { sType = SkillMagic }
			if idx%10 == 0 { sType = SkillBuff }
			if idx%15 == 0 { sType = SkillDebuff }

			s := Skill{
				ID:     fmt.Sprintf("S%d", idx),
				Name:   name,
				Type:   sType,
				Rarity: rarity,
				Power:  1.2 + (0.5 * float64(rarity)),
			}

			// Add unique mechanics based on name/rarity
			if strings.Contains(name, "Sunder") || strings.Contains(name, "Blast") {
				s.IgnoreDef = 0.2 + (0.1 * float64(rarity))
			}
			if strings.Contains(name, "Strike") || strings.Contains(name, "Roar") {
				s.StunChance = 0.05 + (0.05 * float64(rarity))
			}
			if strings.Contains(name, "Touch") || strings.Contains(name, "Plea") {
				s.HealPercent = 0.05 + (0.05 * float64(rarity))
			}

			s.Description = fmt.Sprintf("%s skill with %.1fx power.", s.Type, s.Power)
			allSkills = append(allSkills, s)
			idx++
		}
	}
}

func RandomSkill() Skill {
	return allSkills[rand.Intn(len(allSkills))]
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
