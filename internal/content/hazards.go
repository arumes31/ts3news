package content

import (
	"math"
	"math/rand/v2"
	"strings"
	"ts3news/internal/i18n"
)

// HazardType represents the category of environmental hazard
type HazardType string

const (
	HazardDamageOverTime HazardType = "DamageOverTime"
	HazardStatReduction  HazardType = "StatReduction"
	HazardVisionImpair   HazardType = "VisionImpair"
	HazardMovementImpair HazardType = "MovementImpair"
)

// ZoneType represents the category of zone for hazard compatibility
type ZoneType string

const (
	ZoneVolcanic    ZoneType = "Volcanic"
	ZoneUnderground ZoneType = "Underground"
	ZoneHell        ZoneType = "Hell"
	ZoneSwamp       ZoneType = "Swamp"
	ZoneCave        ZoneType = "Cave"
	ZoneRuins       ZoneType = "Ruins"
	ZoneDesert      ZoneType = "Desert"
	ZoneWasteland   ZoneType = "Wasteland"
	ZoneTundra      ZoneType = "Tundra"
	ZoneMountain    ZoneType = "Mountain"
	ZoneBeach       ZoneType = "Beach"
	ZoneDungeon     ZoneType = "Dungeon"
	ZoneMagic       ZoneType = "Magic"
)

// Hazard represents an environmental hazard in a zone
type Hazard struct {
	ID          string
	Name        string
	Description string
	Type        HazardType
	EffectValue float64    // Percentage or flat value depending on type
	Duration    int        // Number of combat rounds
	ZoneTypes   []ZoneType // Which zone types can have this hazard
	Rarity      float64    // 0.0-1.0 chance of appearing
	Resistance  string     // Stat that provides resistance (e.g., "STA", "INT")
}

// HazardEffect represents an active hazard effect on a combatant
type HazardEffect struct {
	Hazard      Hazard
	Remaining   int
	AppliedTo   string // UID or mob name
	EffectValue float64
}

// AllHazards contains all possible environmental hazards
var AllHazards = []Hazard{
	{
		ID:          "HAZ_LAVA",
		Name:        i18n.T("hazard.boiling_lava"),
		Description: i18n.T("hazard.boiling_lava.desc"),
		Type:        HazardDamageOverTime,
		EffectValue: 0.05, // 5% of max HP per round
		Duration:    3,
		ZoneTypes:   []ZoneType{ZoneVolcanic, ZoneUnderground, ZoneHell},
		Rarity:      0.15,
		Resistance:  "STA",
	},
	{
		ID:          "HAZ_POISON_GAS",
		Name:        i18n.T("hazard.toxic_fumes"),
		Description: i18n.T("hazard.toxic_fumes.desc"),
		Type:        HazardStatReduction,
		EffectValue: 0.30, // 30% stat reduction
		Duration:    4,
		ZoneTypes:   []ZoneType{ZoneSwamp, ZoneCave, ZoneRuins},
		Rarity:      0.20,
		Resistance:  "STA",
	},
	{
		ID:          "HAZ_SANDSTORM",
		Name:        i18n.T("hazard.raging_sandstorm"),
		Description: i18n.T("hazard.raging_sandstorm.desc"),
		Type:        HazardVisionImpair,
		EffectValue: 0.40, // 40% chance to miss attacks
		Duration:    5,
		ZoneTypes:   []ZoneType{ZoneDesert, ZoneWasteland},
		Rarity:      0.25,
		Resistance:  "SPD",
	},
	{
		ID:          "HAZ_BLIZZARD",
		Name:        i18n.T("hazard.howling_blizzard"),
		Description: i18n.T("hazard.howling_blizzard.desc"),
		Type:        HazardMovementImpair,
		EffectValue: 0.20, // 20% speed reduction
		Duration:    4,
		ZoneTypes:   []ZoneType{ZoneTundra, ZoneMountain},
		Rarity:      0.20,
		Resistance:  "STA",
	},
	{
		ID:          "HAZ_RADIATION",
		Name:        i18n.T("hazard.deadly_radiation"),
		Description: i18n.T("hazard.deadly_radiation.desc"),
		Type:        HazardDamageOverTime,
		EffectValue: 0.08, // 8% of max HP per round
		Duration:    5,
		ZoneTypes:   []ZoneType{ZoneWasteland, ZoneRuins, ZoneHell},
		Rarity:      0.10,
		Resistance:  "INT",
	},
	{
		ID:          "HAZ_QUICKSAND",
		Name:        i18n.T("hazard.treacherous_quicksand"),
		Description: i18n.T("hazard.treacherous_quicksand.desc"),
		Type:        HazardMovementImpair,
		EffectValue: 0.30, // 30% speed reduction
		Duration:    3,
		ZoneTypes:   []ZoneType{ZoneSwamp, ZoneBeach},
		Rarity:      0.15,
		Resistance:  "STR",
	},
	{
		ID:          "HAZ_CURSED_AURA",
		Name:        i18n.T("hazard.cursed_aura"),
		Description: i18n.T("hazard.cursed_aura.desc"),
		Type:        HazardStatReduction,
		EffectValue: 0.25, // 25% stat reduction
		Duration:    6,
		ZoneTypes:   []ZoneType{ZoneRuins, ZoneDungeon, ZoneHell},
		Rarity:      0.10,
		Resistance:  "LCK",
	},
	{
		ID:          "HAZ_MAGIC_DRAIN",
		Name:        i18n.T("hazard.arcane_vortex"),
		Description: i18n.T("hazard.arcane_vortex.desc"),
		Type:        HazardStatReduction,
		EffectValue: 0.40, // 40% skill effectiveness reduction
		Duration:    4,
		ZoneTypes:   []ZoneType{ZoneMagic, ZoneRuins, ZoneDungeon},
		Rarity:      0.08,
		Resistance:  "INT",
	},
}

// GetZoneHazards selects appropriate hazards for a zone based on type and difficulty
// getZoneTypeFromName determines the zone type based on zone name
func getZoneTypeFromName(zoneName string) ZoneType {
	zoneName = strings.ToLower(zoneName)

	switch {
	case strings.Contains(zoneName, "volcanic"), strings.Contains(zoneName, "molten"), strings.Contains(zoneName, "lava"), strings.Contains(zoneName, "fire"):
		return ZoneVolcanic
	case strings.Contains(zoneName, "underground"), strings.Contains(zoneName, "cave"), strings.Contains(zoneName, "mine"):
		return ZoneUnderground
	case strings.Contains(zoneName, "hell"), strings.Contains(zoneName, "demon"), strings.Contains(zoneName, "inferno"):
		return ZoneHell
	case strings.Contains(zoneName, "swamp"), strings.Contains(zoneName, "marsh"), strings.Contains(zoneName, "bog"):
		return ZoneSwamp
	case strings.Contains(zoneName, "ruins"), strings.Contains(zoneName, "ancient"):
		return ZoneRuins
	case strings.Contains(zoneName, "desert"), strings.Contains(zoneName, "dune"):
		return ZoneDesert
	case strings.Contains(zoneName, "wasteland"), strings.Contains(zoneName, "scrap"), strings.Contains(zoneName, "radioactive"):
		return ZoneWasteland
	case strings.Contains(zoneName, "tundra"), strings.Contains(zoneName, "arctic"), strings.Contains(zoneName, "frost"):
		return ZoneTundra
	case strings.Contains(zoneName, "mountain"), strings.Contains(zoneName, "peak"), strings.Contains(zoneName, "alpine"):
		return ZoneMountain
	case strings.Contains(zoneName, "beach"), strings.Contains(zoneName, "shore"), strings.Contains(zoneName, "coast"):
		return ZoneBeach
	case strings.Contains(zoneName, "dungeon"), strings.Contains(zoneName, "tomb"), strings.Contains(zoneName, "crypt"):
		return ZoneDungeon
	case strings.Contains(zoneName, "magic"), strings.Contains(zoneName, "arcane"), strings.Contains(zoneName, "spell"):
		return ZoneMagic
	default:
		return ZoneDesert // Default fallback
	}
}

func GetZoneHazards(zone Zone, difficulty float64) []Hazard {
	var applicable []Hazard
	zoneType := getZoneTypeFromName(zone.Name)

	for _, hazard := range AllHazards {
		// Check if hazard is applicable to this zone type
		for _, zt := range hazard.ZoneTypes {
			if zt == zoneType {
				applicable = append(applicable, hazard)
				break
			}
		}
	}

	if len(applicable) == 0 {
		return nil
	}

	// Adjust hazard count based on difficulty (1-3 hazards)
	// difficulty 1.0 -> 1, difficulty 3.0 -> 3
	hazardCount := int(math.Round(difficulty))
	if hazardCount < 1 {
		hazardCount = 1
	}
	if hazardCount > 3 {
		hazardCount = 3
	}

	// Shuffle to ensure uniqueness
	rand.Shuffle(len(applicable), func(i, j int) {
		applicable[i], applicable[j] = applicable[j], applicable[i]
	})

	if hazardCount > len(applicable) {
		hazardCount = len(applicable)
	}

	return applicable[:hazardCount]
}

// ApplyHazardEffects applies hazard effects to all combatants at the start of a round
func ApplyHazardEffects(
	users []*UserInCombat,
	mobs []*Mob,
	hazards []HazardEffect,
	zone Zone,
	logs *[]string,
) []HazardEffect {
	var remainingEffects []HazardEffect

	// Reset temporary modifiers before applying active hazards to prevent compounding
	for _, u := range users {
		u.STRMod = 1.0
		u.DEFMod = 1.0
		u.SPDMod = 1.0
	}
	for _, m := range mobs {
		m.STRMod = 1.0
		m.DEFMod = 1.0
		m.SPDMod = 1.0
	}

	for _, effect := range hazards {
		// Decrement duration
		effect.Remaining--
		if effect.Remaining <= 0 {
			*logs = append(*logs, i18n.T("content.hazard.dissipated", effect.Hazard.Name))
			continue
		}

		switch effect.Hazard.Type {
		case HazardDamageOverTime:
			// Apply damage to all combatants
			for _, u := range users {
				if u.CurrentHP <= 0 {
					continue
				}
				damage := int(float64(u.Stats.HP) * effect.Hazard.EffectValue)
				if damage < 1 {
					damage = 1
				}
				// Apply resistance
				resistance := getResistanceValue(u.Stats, effect.Hazard.Resistance)
				damage = int(float64(damage) * (1.0 - resistance))
				u.CurrentHP -= damage
				*logs = append(*logs, i18n.T("content.hazard.damage", u.Nickname, damage, effect.Hazard.Name))
			}
			for _, m := range mobs {
				if m.CurrentHP <= 0 {
					continue
				}
				damage := int(float64(m.MaxHP) * effect.Hazard.EffectValue)
				if damage < 1 {
					damage = 1
				}
				// Mobs also have resistance
				resistance := getResistanceValue(m.Stats, effect.Hazard.Resistance)
				damage = int(float64(damage) * (1.0 - resistance))
				m.CurrentHP -= damage
				*logs = append(*logs, i18n.T("content.hazard.damage", m.Name, damage, effect.Hazard.Name))
			}
		case HazardStatReduction:
			// Apply stat reduction to all combatants via modifiers
			for _, u := range users {
				if u.CurrentHP <= 0 {
					continue
				}
				resistance := getResistanceValue(u.Stats, effect.Hazard.Resistance)
				reduction := effect.Hazard.EffectValue * (1.0 - resistance)
				// Apply to temporary modifiers instead of base stats
				u.STRMod *= (1.0 - reduction)
				u.DEFMod *= (1.0 - reduction)
				u.SPDMod *= (1.0 - reduction)
				*logs = append(*logs, i18n.T("content.hazard.weakened", u.Nickname, effect.Hazard.Name, reduction*100))
			}
			for _, m := range mobs {
				if m.CurrentHP <= 0 {
					continue
				}
				// Mobs also have resistance now
				resistance := getResistanceValue(m.Stats, effect.Hazard.Resistance)
				reduction := effect.Hazard.EffectValue * (1.0 - resistance)
				m.STRMod *= (1.0 - reduction)
				m.DEFMod *= (1.0 - reduction)
				m.SPDMod *= (1.0 - reduction)
				*logs = append(*logs, i18n.T("content.hazard.weakened", m.Name, effect.Hazard.Name, reduction*100))
			}
		case HazardVisionImpair:
			// Apply miss chance to users
			for _, u := range users {
				if u.CurrentHP <= 0 {
					continue
				}
				resistance := getResistanceValue(u.Stats, effect.Hazard.Resistance)
				impairment := effect.Hazard.EffectValue * (1.0 - resistance)
				// This will be checked during attack rolls
				*logs = append(*logs, i18n.T("content.hazard.vision_impaired", u.Nickname, effect.Hazard.Name, impairment*100))
			}
		case HazardMovementImpair:
			// Apply speed reduction to users via modifiers
			for _, u := range users {
				if u.CurrentHP <= 0 {
					continue
				}
				resistance := getResistanceValue(u.Stats, effect.Hazard.Resistance)
				reduction := effect.Hazard.EffectValue * (1.0 - resistance)
				u.SPDMod *= (1.0 - reduction)
				*logs = append(*logs, i18n.T("content.hazard.movement_impaired", u.Nickname, effect.Hazard.Name, reduction*100))
			}
		}

		remainingEffects = append(remainingEffects, effect)
	}

	return remainingEffects
}

// getResistanceValue calculates resistance value from stats (0.0-0.75)
func getResistanceValue(stats Stats, resistanceStat string) float64 {
	var statValue int
	switch resistanceStat {
	case "STR":
		statValue = stats.STR
	case "DEF":
		statValue = stats.DEF
	case "SPD":
		statValue = stats.SPD
	case "LCK":
		statValue = stats.LCK
	case "INT":
		statValue = stats.INT
	case "STA":
		statValue = stats.STA
	default:
		statValue = 0
	}

	// Resistance ranges from 0% to 75% based on stat value
	resistance := float64(statValue) / 2000.0
	if resistance > 0.75 {
		resistance = 0.75
	}
	return resistance
}

// GetHazardProtectionGear returns gear that provides protection against hazards
// HazardGear represents protective gear against environmental hazards
type HazardGear struct {
	Name        string
	Description string
	Protection  string // Main stat protected (e.g., "STA", "INT", "SPD")
	Rarity      string
}

// HazardConsumable represents consumables that protect against hazards
type HazardConsumable struct {
	Name        string
	Description string
	Type        ConsumableType
	EffectStat  string  // Main stat affected (e.g., "STA", "INT", "SPD")
	EffectValue float64 // Percentage boost
	Duration    int     // rounds
}

var hazardProtectionGear = []HazardGear{
	{Name: "Heat-Resistant Plate", Description: "Protects against extreme heat", Protection: "STA", Rarity: "Rare"},
	{Name: "Fireproof Cloak", Description: "Reduces fire damage", Protection: "STA", Rarity: "Rare"},
	{Name: "Molten Core Gauntlets", Description: "Heat-resistant gloves", Protection: "STA", Rarity: "Epic"},
	{Name: "Gas Mask", Description: "Protects against poisonous gases", Protection: "STA", Rarity: "Uncommon"},
	{Name: "Antitoxin Armor", Description: "Reduces poison effects", Protection: "STA", Rarity: "Rare"},
	{Name: "Desert Goggles", Description: "Improves vision in sandstorms", Protection: "SPD", Rarity: "Uncommon"},
	{Name: "Sandstorm Cloak", Description: "Reduces sandstorm effects", Protection: "SPD", Rarity: "Rare"},
	{Name: "Protective Ward", Description: "General protection against hazards", Protection: "DEF", Rarity: "Common"},
	{Name: "Resistant Tunic", Description: "General hazard resistance", Protection: "DEF", Rarity: "Common"},
	{Name: "Arcane Shield", Description: "Protects against magic-based hazards", Protection: "INT", Rarity: "Rare"},
}

var hazardProtectionConsumables = []HazardConsumable{
	{Name: "Health Potion", Description: "Restores health", Type: ConsumableHealing, EffectStat: "HP", EffectValue: 0.3, Duration: 0},
	{Name: "Stamina Elixir", Description: "Boosts stamina", Type: ConsumableBuff, EffectStat: "STA", EffectValue: 0.2, Duration: 3},
	{Name: "Speed Potion", Description: "Increases speed", Type: ConsumableBuff, EffectStat: "SPD", EffectValue: 0.3, Duration: 3},
	{Name: "Intellect Draught", Description: "Boosts intelligence", Type: ConsumableBuff, EffectStat: "INT", EffectValue: 0.25, Duration: 3},
	{Name: "Antidote", Description: "Cures poison", Type: ConsumableHealing, EffectStat: "HP", EffectValue: 0.5, Duration: 0},
	{Name: "Clarity Potion", Description: "Improves vision", Type: ConsumableBuff, EffectStat: "SPD", EffectValue: 0.4, Duration: 3},
}

// GetHazardProtectionGear returns gear that provides protection against specific hazards.
// The gear is selected based on hazard type and resistance properties.
func GetHazardProtectionGear(hazard Hazard) []HazardGear {
	// Pre-allocate slice for better performance
	protectionGear := make([]HazardGear, 0, 3)

	// Map hazard types to gear protection stats
	hazardToProtection := map[HazardType][]string{
		HazardDamageOverTime: {"STA", "DEF"}, // Heat, radiation, etc.
		HazardStatReduction:  {"STA", "INT"}, // Poison, curses, etc.
		HazardVisionImpair:   {"SPD", "LCK"}, // Sandstorms, darkness
		HazardMovementImpair: {"SPD", "STR"}, // Quicksand, blizzards
	}

	// Get protection stats for this hazard type
	protectionStats := hazardToProtection[hazard.Type]
	if protectionStats == nil {
		// Default to general protection for unknown hazard types
		protectionStats = []string{"DEF", "STA"}
	}

	// Filter gear by protection relevance
	for _, gear := range hazardProtectionGear {
		// Check if gear protects against this hazard's resistance stat
		if gear.Protection == hazard.Resistance {
			protectionGear = append(protectionGear, gear)
			continue
		}

		// Check if gear protects against any of the hazard's protection stats
		for _, stat := range protectionStats {
			if gear.Protection == stat {
				protectionGear = append(protectionGear, gear)
				break
			}
		}
	}

	// Add hazard-specific gear based on ID
	switch hazard.ID {
	case "HAZ_LAVA", "HAZ_RADIATION":
		for _, gear := range hazardProtectionGear {
			if strings.Contains(strings.ToLower(gear.Name), "heat") ||
				strings.Contains(strings.ToLower(gear.Name), "fire") {
				protectionGear = append(protectionGear, gear)
			}
		}
	case "HAZ_POISON_GAS":
		for _, gear := range hazardProtectionGear {
			if strings.Contains(strings.ToLower(gear.Name), "gas mask") ||
				strings.Contains(strings.ToLower(gear.Name), "antitoxin") {
				protectionGear = append(protectionGear, gear)
			}
		}
	case "HAZ_SANDSTORM":
		for _, gear := range hazardProtectionGear {
			if strings.Contains(strings.ToLower(gear.Name), "goggles") ||
				strings.Contains(strings.ToLower(gear.Name), "visor") {
				protectionGear = append(protectionGear, gear)
			}
		}
	case "HAZ_BLIZZARD":
		for _, gear := range hazardProtectionGear {
			if strings.Contains(strings.ToLower(gear.Name), "thermal") ||
				strings.Contains(strings.ToLower(gear.Name), "insulated") {
				protectionGear = append(protectionGear, gear)
			}
		}
	}

	// Remove duplicates while preserving order
	protectionGear = removeDuplicateGear(protectionGear)

	// If no specific gear found, return general protective gear
	if len(protectionGear) == 0 {
		for _, gear := range hazardProtectionGear {
			if strings.Contains(strings.ToLower(gear.Name), "protective") ||
				strings.Contains(strings.ToLower(gear.Name), "resistant") {
				protectionGear = append(protectionGear, gear)
			}
		}
	}

	// Return up to 3 most relevant pieces of gear
	if len(protectionGear) > 3 {
		// Prioritize gear that matches the hazard's resistance stat
		protectionGear = prioritizeGearByResistance(protectionGear, hazard.Resistance)
		return protectionGear[:3]
	}

	return protectionGear
}

// removeDuplicateGear removes duplicate gear items while preserving order
func removeDuplicateGear(gear []HazardGear) []HazardGear {
	seen := make(map[string]bool)
	uniqueGear := make([]HazardGear, 0, len(gear))

	for _, g := range gear {
		if !seen[g.Name] {
			seen[g.Name] = true
			uniqueGear = append(uniqueGear, g)
		}
	}

	return uniqueGear
}

// prioritizeGearByResistance prioritizes gear that matches the specified resistance stat
func prioritizeGearByResistance(gear []HazardGear, resistanceStat string) []HazardGear {
	// Separate gear into matching and non-matching
	var matching []HazardGear
	var nonMatching []HazardGear

	for _, g := range gear {
		if g.Protection == resistanceStat {
			matching = append(matching, g)
		} else {
			nonMatching = append(nonMatching, g)
		}
	}

	// Combine with matching gear first
	return append(matching, nonMatching...)
}

// GetHazardProtectionConsumable returns consumables that mitigate specific hazard effects.
// The consumables are selected based on hazard type and resistance properties.
func GetHazardProtectionConsumable(hazard Hazard) []HazardConsumable {
	// Pre-allocate slice for better performance
	protection := make([]HazardConsumable, 0, 2)

	// Map hazard types to consumable effect stats
	hazardToEffectStats := map[HazardType][]string{
		HazardDamageOverTime: {"HP", "STA"},  // Healing, stamina
		HazardStatReduction:  {"STA", "INT"}, // Stamina, intelligence
		HazardVisionImpair:   {"SPD", "LCK"}, // Speed, luck
		HazardMovementImpair: {"SPD", "STR"}, // Speed, strength
	}

	// Get effect stats for this hazard type
	effectStats := hazardToEffectStats[hazard.Type]
	if effectStats == nil {
		// Default to general buffs for unknown hazard types
		effectStats = []string{"HP", "STA"}
	}

	// Filter consumables by effect relevance
	for _, cons := range hazardProtectionConsumables {
		// Healing consumables are always useful for damage hazards
		if hazard.Type == HazardDamageOverTime && cons.Type == ConsumableHealing {
			protection = append(protection, cons)
			continue
		}

		// Check if consumable affects any of the hazard's effect stats
		for _, stat := range effectStats {
			if cons.EffectStat == stat {
				protection = append(protection, cons)
				break
			}
		}
	}

	// Add hazard-specific consumables based on ID
	switch hazard.ID {
	case "HAZ_LAVA", "HAZ_RADIATION":
		for _, cons := range hazardProtectionConsumables {
			if strings.Contains(strings.ToLower(cons.Name), "heat") ||
				strings.Contains(strings.ToLower(cons.Name), "fire") {
				protection = append(protection, cons)
			}
		}
	case "HAZ_POISON_GAS":
		for _, cons := range hazardProtectionConsumables {
			if strings.Contains(strings.ToLower(cons.Name), "antidote") ||
				strings.Contains(strings.ToLower(cons.Name), "cure") {
				protection = append(protection, cons)
			}
		}
	case "HAZ_SANDSTORM":
		for _, cons := range hazardProtectionConsumables {
			if strings.Contains(strings.ToLower(cons.Name), "clarity") ||
				strings.Contains(strings.ToLower(cons.Name), "vision") {
				protection = append(protection, cons)
			}
		}
	case "HAZ_BLIZZARD":
		for _, cons := range hazardProtectionConsumables {
			if strings.Contains(strings.ToLower(cons.Name), "warmth") ||
				strings.Contains(strings.ToLower(cons.Name), "thermal") {
				protection = append(protection, cons)
			}
		}
	}

	// Remove duplicates while preserving order
	protection = removeDuplicateConsumables(protection)

	// If no specific consumables found, return general buffs/healing
	if len(protection) == 0 {
		for _, cons := range hazardProtectionConsumables {
			if cons.Type == ConsumableBuff || cons.Type == ConsumableHealing {
				protection = append(protection, cons)
			}
		}
	}

	// Return up to 2 most relevant consumables
	if len(protection) > 2 {
		// Prioritize consumables that match the hazard's resistance stat
		protection = prioritizeConsumablesByStat(protection, hazard.Resistance)
		return protection[:2]
	}

	return protection
}

// removeDuplicateConsumables removes duplicate consumables while preserving order
func removeDuplicateConsumables(consumables []HazardConsumable) []HazardConsumable {
	seen := make(map[string]bool)
	uniqueConsumables := make([]HazardConsumable, 0, len(consumables))

	for _, c := range consumables {
		if !seen[c.Name] {
			seen[c.Name] = true
			uniqueConsumables = append(uniqueConsumables, c)
		}
	}

	return uniqueConsumables
}

// prioritizeConsumablesByStat prioritizes consumables that affect the specified stat
func prioritizeConsumablesByStat(consumables []HazardConsumable, stat string) []HazardConsumable {
	// Separate consumables into matching and non-matching
	var matching []HazardConsumable
	var nonMatching []HazardConsumable

	for _, c := range consumables {
		if c.EffectStat == stat {
			matching = append(matching, c)
		} else {
			nonMatching = append(nonMatching, c)
		}
	}

	// Combine with matching consumables first
	return append(matching, nonMatching...)
}
