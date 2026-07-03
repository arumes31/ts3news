package content

import (
	"math/rand/v2"
	"strings"
	"ts3news/internal/i18n"
)

// StealthType represents different stealth mechanics
type StealthType string

// Stealth mechanic kinds.
const (
	StealthPassive     StealthType = "Passive"     // Always-on stealth bonus
	StealthActive      StealthType = "Active"      // Requires activation
	StealthSituational StealthType = "Situational" // Triggered by conditions
)

// StealthEffect represents a stealth bonus or ability
type StealthEffect struct {
	ID          string
	Name        string
	Description string
	Type        StealthType
	EffectValue float64 // Percentage bonus/penalty
	Duration    int     // Rounds, 0 for passive
	Cooldown    int     // Rounds
	Requires    string  // Required gear/skill (e.g., "Night Cloak")
}

// StealthState tracks a user's stealth status during combat
type StealthState struct {
	CurrentStealth  float64 // 0.0-1.0 (0% to 100% stealth)
	DetectionChance float64 // 0.0-1.0 (chance mobs detect you)
	ActiveEffects   []StealthEffect
	Cooldowns       map[string]int // Track cooldowns by effect ID
}

// StealthDetection represents a mob's ability to detect stealthed players
type StealthDetection struct {
	BaseDetection  float64 // 0.0-1.0 (base chance to detect stealthed players)
	Perception     float64 // Bonus to detection based on mob level/stats
	SituationalMod float64 // Bonus from external factors (light, sound, etc.)
}

// AllStealthEffects contains all available stealth abilities and bonuses
var AllStealthEffects = []StealthEffect{
	{
		ID:          "STEALTH_BASIC",
		Name:        i18n.T("content.stealth.natural_camouflage.name"),
		Description: i18n.T("content.stealth.natural_camouflage.description"),
		Type:        StealthPassive,
		EffectValue: 0.15, // 15% stealth bonus
		Duration:    0,
		Cooldown:    0,
	},
	{
		ID:          "STEALTH_CLOAK",
		Name:        i18n.T("content.stealth.cloak_of_shadows.name"),
		Description: i18n.T("content.stealth.cloak_of_shadows.description"),
		Type:        StealthPassive,
		EffectValue: 0.25, // 25% stealth bonus
		Duration:    0,
		Cooldown:    0,
		Requires:    "Shadow Cloak",
	},
	{
		ID:          "STEALTH_NIGHT",
		Name:        i18n.T("content.stealth.night_stalker.name"),
		Description: i18n.T("content.stealth.night_stalker.description"),
		Type:        StealthSituational,
		EffectValue: 0.40, // 40% stealth bonus
		Duration:    0,
		Cooldown:    0,
	},
	{
		ID:          "STEALTH_AMBUSH",
		Name:        i18n.T("content.stealth.ambush_predator.name"),
		Description: i18n.T("content.stealth.ambush_predator.description"),
		Type:        StealthActive,
		EffectValue: 0.50, // 50% bonus damage
		Duration:    1,
		Cooldown:    5,
	},
	{
		ID:          "STEALTH_DISTRACT",
		Name:        i18n.T("content.stealth.misdirection.name"),
		Description: i18n.T("content.stealth.misdirection.description"),
		Type:        StealthActive,
		EffectValue: -0.30, // Reduces detection chance by 30%
		Duration:    3,
		Cooldown:    8,
	},
	{
		ID:          "STEALTH_SILENT",
		Name:        i18n.T("content.stealth.silent_movement.name"),
		Description: i18n.T("content.stealth.silent_movement.description"),
		Type:        StealthActive,
		EffectValue: -0.40, // Reduces detection chance by 40%
		Duration:    4,
		Cooldown:    6,
	},
}

// CalculateStealth calculates a user's current stealth level
func CalculateStealth(user *UserInCombat, zone Zone, timeOfDay string) StealthState {
	state := StealthState{
		CurrentStealth:  0.0,
		DetectionChance: 0.5, // Base 50% detection chance
		ActiveEffects:   []StealthEffect{},
		Cooldowns:       make(map[string]int),
	}

	// Apply gear-based stealth bonuses (check gear names/special from equipped items)
	gearBonus := 0.0
	for _, gear := range user.Equipped {
		if gear.Special == EffectStealth {
			gearBonus += 0.10
		}
		if strings.Contains(strings.ToLower(gear.Name), "shadow") ||
			strings.Contains(strings.ToLower(gear.Name), "cloak") ||
			strings.Contains(strings.ToLower(gear.Name), "stealth") {
			gearBonus += 0.05
		}
	}
	state.CurrentStealth += gearBonus

	// Apply skill-based stealth bonuses
	for _, skill := range user.Skills {
		if strings.Contains(strings.ToLower(skill.Name), "stealth") ||
			strings.Contains(strings.ToLower(skill.Name), "sneak") {
			state.CurrentStealth += 0.15 // 15% bonus per stealth skill
		}
	}

	// Apply passive stealth effects
	for _, effect := range AllStealthEffects {
		if effect.Type == StealthPassive {
			// Check if user has required gear
			if effect.Requires == "" || hasRequiredGear(user, effect.Requires) {
				state.CurrentStealth += effect.EffectValue
				state.ActiveEffects = append(state.ActiveEffects, effect)
			}
		}
	}

	// Apply situational effects (night time bonus)
	if timeOfDay == "night" {
		for _, effect := range AllStealthEffects {
			if effect.ID == "STEALTH_NIGHT" {
				state.CurrentStealth += effect.EffectValue
				state.ActiveEffects = append(state.ActiveEffects, effect)
			}
		}
	}

	// Apply zone modifiers (forests, shadows, etc. provide bonuses)
	zoneBonus := getZoneStealthBonus(zone)
	state.CurrentStealth += zoneBonus

	// Ensure stealth is capped at 90% (never 100%)
	if state.CurrentStealth > 0.9 {
		state.CurrentStealth = 0.9
	}

	// Calculate detection chance (inverse of stealth)
	state.DetectionChance = 0.5 * (1.0 - state.CurrentStealth)

	return state
}

// CalculateMobDetection calculates a mob's ability to detect stealthed players
func CalculateMobDetection(mob *Mob, zone Zone, timeOfDay string) StealthDetection {
	detection := StealthDetection{
		BaseDetection:  0.3, // Base 30% detection chance
		Perception:     0.0,
		SituationalMod: 0.0,
	}

	// Perception scales with mob level and stats
	detection.Perception = float64(mob.Level) * 0.01
	if mob.Stats.INT > 50 {
		detection.Perception += float64(mob.Stats.INT) * 0.002
	}

	// Situational modifiers
	if timeOfDay == "night" {
		detection.SituationalMod -= 0.1 // Harder to see at night
	} else {
		detection.SituationalMod += 0.1 // Easier to see during day
	}

	// Zone modifiers
	zoneMod := getZoneDetectionModifier(zone)
	detection.SituationalMod += zoneMod

	return detection
}

// CheckStealthDetection determines if a mob detects a stealthed player
func CheckStealthDetection(userStealth StealthState, mobDetection StealthDetection) bool {
	totalDetectionChance := mobDetection.BaseDetection +
		mobDetection.Perception +
		mobDetection.SituationalMod +
		userStealth.DetectionChance

	// Ensure detection chance is between 5% and 95%
	if totalDetectionChance < 0.05 {
		totalDetectionChance = 0.05
	}
	if totalDetectionChance > 0.95 {
		totalDetectionChance = 0.95
	}

	// Roll for detection
	// #nosec G404
	roll := rand.Float64() // #nosec G404
	return roll <= totalDetectionChance
}

// ApplyStealthAttack applies stealth-based combat advantages
func ApplyStealthAttack(_ *UserInCombat, _ *Mob, stealthState StealthState, detected bool) float64 {
	bonusDamage := 0.0
	undetected := !detected

	// Check for active ambush effects
	for _, effect := range stealthState.ActiveEffects {
		if effect.ID == "STEALTH_AMBUSH" && undetected {
			bonusDamage += effect.EffectValue
		}
	}

	// If undetected, apply first strike bonus
	if undetected {
		bonusDamage += 0.25 // 25% base first strike bonus
	}

	return bonusDamage
}

// GetStealthGear returns gear that enhances stealth (placeholder implementation)
func GetStealthGear() []HazardGear {
	var stealthGear []HazardGear
	stealthGearNames := []string{i18n.T("content.stealth.gear.shadow_cloak"), i18n.T("content.stealth.gear.night_cloak"), i18n.T("content.stealth.gear.stealth_tunic"), i18n.T("content.stealth.gear.assassins_garb")}

	for _, gearName := range stealthGearNames {
		stealthGear = append(stealthGear, HazardGear{
			Name:        gearName,
			Description: i18n.T("content.stealth.gear.enhances_stealth"),
			Protection:  "STEALTH",
			Rarity:      "Rare",
		})
	}
	return stealthGear
}

// GetStealthConsumables returns consumables that enhance stealth (placeholder implementation)
func GetStealthConsumables() []HazardConsumable {
	return []HazardConsumable{
		{
			Name:        i18n.T("content.stealth.consumable.shadow_potion"),
			Description: i18n.T("content.stealth.consumable.shadow_potion_desc"),
			Type:        ConsumableBuff,
			EffectStat:  "STEALTH",
			EffectValue: 0.3,
			Duration:    3,
		},
		{
			Name:        i18n.T("content.stealth.consumable.cloak_elixir"),
			Description: i18n.T("content.stealth.consumable.cloak_elixir_desc"),
			Type:        ConsumableBuff,
			EffectStat:  "STEALTH",
			EffectValue: 0.25,
			Duration:    4,
		},
	}
}

// hasRequiredGear checks if user has required gear for a stealth effect (placeholder)
func hasRequiredGear(user *UserInCombat, requiredGear string) bool {
	for _, gear := range user.Equipped {
		if strings.Contains(strings.ToLower(gear.Name), strings.ToLower(requiredGear)) {
			return true
		}
	}
	return false
}

// getZoneStealthBonus returns stealth bonus based on zone type
func getZoneStealthBonus(zone Zone) float64 {
	zoneName := strings.ToLower(zone.Name)

	if strings.Contains(zoneName, "forest") || strings.Contains(zoneName, "wood") {
		return 0.2 // 20% bonus in forests
	}
	if strings.Contains(zoneName, "shadow") || strings.Contains(zoneName, "dark") {
		return 0.25 // 25% bonus in dark zones
	}
	if strings.Contains(zoneName, "ruin") || strings.Contains(zoneName, "abandon") {
		return 0.15 // 15% bonus in ruins
	}
	if strings.Contains(zoneName, "urban") || strings.Contains(zoneName, "city") {
		return -0.1 // 10% penalty in urban areas
	}

	return 0.0
}

// getZoneDetectionModifier returns detection modifier based on zone type
func getZoneDetectionModifier(zone Zone) float64 {
	zoneName := strings.ToLower(zone.Name)

	if strings.Contains(zoneName, "forest") || strings.Contains(zoneName, "wood") {
		return -0.1 // 10% harder to detect in forests
	}
	if strings.Contains(zoneName, "plains") || strings.Contains(zoneName, "open") {
		return 0.2 // 20% easier to detect in open areas
	}
	if strings.Contains(zoneName, "urban") || strings.Contains(zoneName, "city") {
		return 0.15 // 15% easier to detect in cities
	}

	return 0.0
}
