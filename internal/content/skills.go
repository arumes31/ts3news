package content

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"ts3news/internal/i18n"
)

type SkillType string

const (
	SkillPhysical SkillType = "Physical"
	SkillMagic    SkillType = "Magic"
	SkillBuff     SkillType = "Buff"
	SkillDebuff   SkillType = "Debuff"
	SkillUltimate SkillType = "Ultimate"
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

// UltimateSkill represents a powerful ability with multi-round cooldown
type UltimateSkill struct {
	ID              string
	Name            string
	Rarity          Rarity
	Power           float64 // Damage multiplier (e.g., 5.0 = 5x normal damage)
	CooldownRounds  int     // Total rounds to wait after use
	CurrentCooldown int     // Current cooldown counter (0 = ready)
	Description     string
	Special         ItemEffect
}

var allSkills []Skill
var skillsInitOnce sync.Once

func initSkills() {
	skillsInitOnce.Do(func() {
		// Inspired Prefix & Action pools for 1500+ variants
		prefixes := i18n.Pool("pool.skill.prefix")
		actions := i18n.Pool("pool.skill.action")

		// Safety check for empty pools
		if len(prefixes) == 0 {
			prefixes = []string{"Fiery", "Icy", "Shadow", "Holy", "Arcane", "Toxic", "Storm", "Earth", "Wind", "Blood"}
		}
		if len(actions) == 0 {
			actions = []string{"Strike", "Blast", "Bolt", "Heal", "Shield", "Curse", "Sunder", "Bash", "Mend", "Drain"}
		}

		// Add basic novice skills
		allSkills = append(allSkills, Skill{
			ID:          "S0_1",
			Name:        "Novice Spark",
			Type:        SkillMagic,
			Rarity:      RarityCommon,
			Power:       1.1,
			Description: "A weak magical spark.",
		})
		allSkills = append(allSkills, Skill{
			ID:          "S0_2",
			Name:        "Novice Punch",
			Type:        SkillPhysical,
			Rarity:      RarityCommon,
			Power:       1.1,
			Description: "A basic physical punch.",
		})

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
					if strings.Contains(name, "Bolt") || strings.Contains(name, "Blast") || strings.Contains(name, "Nova") {
						sType = SkillMagic
					}
					if strings.Contains(name, "Heal") || strings.Contains(name, "Mend") || strings.Contains(name, "Shield") {
						sType = SkillBuff
					}
					if strings.Contains(name, "Curse") || strings.Contains(name, "Sunder") || strings.Contains(name, "Drain") {
						sType = SkillDebuff
					}

					s := Skill{
						ID:     fmt.Sprintf("S%d_%d", idx, rIdx),
						Name:   name,
						Type:   sType,
						Rarity: rarity,
						Power:  1.2 + (0.6 * float64(rarity)),
					}

					// Mechanics
					if strings.Contains(name, "Sunder") || strings.Contains(name, "Execute") {
						s.IgnoreDef = 0.3 + (0.1 * float64(rarity))
					}
					if strings.Contains(name, "Bash") || strings.Contains(name, "Shock") {
						s.StunChance = 0.1 + (0.05 * float64(rarity))
					}
					if strings.Contains(name, "Heal") || strings.Contains(name, "Mend") {
						s.HealPercent = 0.1 + (0.05 * float64(rarity))
					}

					// Rare special effects
					// #nosec G404
					if rarity >= RarityEpic && rand.Float64() < 0.1 { // #nosec G404
						s.Special = EffectMindControl
					}
					// #nosec G404
					if rarity == RarityLegendary && rand.Float64() < 0.05 { // #nosec G404
						s.Special = EffectPhoenix
					}

					s.Description = i18n.T("content.skill.description.format", i18n.R(int(s.Rarity)), s.Name, int(rarity)+1)
					allSkills = append(allSkills, s)
				}
				idx++
			}
		}
	})
}

func (s Skill) Score() int {
	score := int(s.Power*100) + int(s.IgnoreDef*100) + int(s.StunChance*100) + int(s.HealPercent*100)
	if s.Special == EffectMindControl {
		score += 500
	}
	if s.Special == EffectPhoenix {
		score += 1000
	}
	return score
}

func RandomSkill() Skill {
	initSkills()
	// #nosec G404
	s := allSkills[rand.IntN(len(allSkills))] // #nosec G404
	// Roll for additional effect if it doesn't have one
	if s.Special == EffectNone {
		s.Special = RandomItemEffect()
	}
	return s
}

func GetSkillByID(id string) (Skill, bool) {
	initSkills()
	for _, s := range allSkills {
		if s.ID == id {
			return s, true
		}
	}
	return Skill{}, false
}

func IsSkill(name string) bool {
	initSkills()
	for _, s := range allSkills {
		if s.Name == name {
			return true
		}
	}
	return false
}

// Ultimate Skill name generation (50 verbs × 20 nouns = 1000 combinations)
var ultimateVerbs = []string{
	"Annihilating", "Devastating", "Obliterating", "Shattering", "Eradicating",
	"Decimating", "Destroying", "Crushing", "Smashing", "Pulverizing",
	"Vaporizing", "Incinerating", "Freezing", "Electrifying", "Corrupting",
	"Purifying", "Banishing", "Summoning", "Channeling", "Unleashing",
	"Igniting", "Extinguishing", "Rending", "Cleaving", "Piercing",
	"Shredding", "Blasting", "Bombarding", "Storming", "Ravaging",
	"Consuming", "Devouring", "Absorbing", "Reflecting", "Amplifying",
	"Nullifying", "Silencing", "Blinding", "Stunning", "Paralyzing",
	"Poisoning", "Cursing", "Blessing", "Healing", "Shielding",
	"Empowering", "Enraging", "Terrifying", "Inspiring", "Transcending",
}
var ultimateNouns = []string{
	"Strike", "Blast", "Wave", "Storm", "Fury",
	"Wrath", "Rage", "Nova", "Burst", "Flare",
	"Surge", "Pulse", "Beam", "Ray", "Bolt",
	"Slash", "Thrust", "Barrage", "Volley", "Onslaught",
}

var allUltimateSkills []UltimateSkill
var ultimateInitOnce sync.Once

// initUltimateSkills lazily builds the ultimate-skill pool on first access. A
// package init() would run before i18n is loaded, baking raw translation keys
// (e.g. "content.skill.ultimate.description.format") into every description; a
// once-guarded lazy build instead resolves the descriptions in the active locale.
func initUltimateSkills() {
	ultimateInitOnce.Do(func() {
		// Generate 1000 unique ultimate skills using deterministic RNG like artifacts.go
		// Use a fixed seed for procedural generation to ensure UltimateSkill IDs (ULT_1, ULT_2...)
		// are stable across bot restarts/rebuilds.
		r := rand.New(rand.NewPCG(42, 42)) // #nosec G404

		idx := 1
		for _, v := range ultimateVerbs {
			for _, n := range ultimateNouns {
				name := v + " " + n

				// Determine rarity (ultimate skills are inherently rare)
				rr := r.Float64()
				var rarity Rarity
				switch {
				case rr < 0.50:
					rarity = RarityRare
				case rr < 0.80:
					rarity = RarityEpic
				case rr < 0.95:
					rarity = RarityLegendary
				case rr < 0.99:
					rarity = RarityMythic
				default:
					rarity = RarityDivine
				}

				// Power scales with rarity
				rarityMult := float64(rarity+1) * 0.5 // Rare=1.5, Epic=2.0, Legendary=2.5, Mythic=3.0, Divine=3.5
				power := 4.0 * rarityMult

				// Cooldown scales with rarity (higher rarity = longer cooldown but more power)
				cooldown := 5 + int(rarity)*2 // Rare=9, Epic=11, Legendary=13, Mythic=15, Divine=17

				skill := UltimateSkill{
					ID:              fmt.Sprintf("ULT_%d", idx),
					Name:            name,
					Rarity:          rarity,
					Power:           power,
					CooldownRounds:  cooldown,
					CurrentCooldown: 0,
					Description:     i18n.T("content.skill.ultimate.description.format", i18n.R(int(rarity)), power, cooldown),
				}
				allUltimateSkills = append(allUltimateSkills, skill)
				idx++
			}
		}

		// Task 66: Revival Ultimate Skill
		allUltimateSkills = append(allUltimateSkills, UltimateSkill{
			ID:             "ULT_REVIVAL",
			Name:           "Divine Revival",
			Rarity:         RarityDivine,
			Power:          0.0,
			CooldownRounds: 15,
			Description:    "Cheat death — revive once per battle at half health.",
			Special:        EffectPhoenix,
		})
	})
}

// RandomUltimateSkill returns a random ultimate skill
func RandomUltimateSkill() UltimateSkill {
	initUltimateSkills()
	// #nosec G404
	return allUltimateSkills[rand.IntN(len(allUltimateSkills))] // #nosec G404
}

// GetUltimateSkillByID returns an ultimate skill by ID
func GetUltimateSkillByID(id string) (UltimateSkill, bool) {
	initUltimateSkills()
	for _, s := range allUltimateSkills {
		if s.ID == id {
			return s, true
		}
	}
	return UltimateSkill{}, false
}

// IsUltimateSkill checks if a name is an ultimate skill
func IsUltimateSkill(name string) bool {
	initUltimateSkills()
	for _, s := range allUltimateSkills {
		if s.Name == name {
			return true
		}
	}
	return false
}
