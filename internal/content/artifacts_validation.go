package content

import (
	"fmt"
	"strings"
)

var itemEffectDescriptions = map[ItemEffect]string{
	EffectThorns:         "Reflects 10% of damage taken",
	EffectVampiric:       "Heals for 5% of damage dealt",
	EffectBerserk:        "+20% STR while below 50% HP",
	EffectLucky:          "+10% Luck",
	EffectTreasureHunter: "+5% item find chance",
	EffectQuick:          "+10% Speed",
	EffectBulwark:        "+10% Defense",
	EffectRadiant:        "+10% XP gained",
	EffectFragile:        "+30% STR but double durability loss",
	EffectSteady:         "-50% stun chance",
	EffectMindControl:    "Chance to capture low-health mobs",
	EffectRegenStack:     "Permanent regen stack on victory",
	EffectPhoenix:        "Revive once per fight at 50% HP",
	EffectStealth:        "Skip first-round mob damage",
	EffectParry:          "10% chance to negate a hit and counter",
	EffectCleanse:        "Removes a negative effect each turn",
	EffectExecutioner:    "+25% damage to targets below 30% HP",
	EffectFocused:        "+10% Crit Rate",
}

var allItemEffects = []ItemEffect{
	EffectThorns,
	EffectVampiric,
	EffectBerserk,
	EffectLucky,
	EffectTreasureHunter,
	EffectQuick,
	EffectBulwark,
	EffectRadiant,
	EffectFragile,
	EffectSteady,
	EffectMindControl,
	EffectRegenStack,
	EffectPhoenix,
	EffectStealth,
	EffectParry,
	EffectCleanse,
	EffectExecutioner,
	EffectFocused,
}

// ItemEffectDescription returns the canonical player-facing explanation for an
// item effect. EffectNone intentionally has no description.
func ItemEffectDescription(effect ItemEffect) string {
	return itemEffectDescriptions[effect]
}

// ValidateGearCatalog checks the integrity rules required by Abyss loot and
// forge presentation before the web server starts.
func ValidateGearCatalog() error {
	return validateGearCatalog(allGear, allItemEffects)
}

func validateGearCatalog(gearCatalog []Gear, effects []ItemEffect) error {
	seen := make(map[string]struct{}, len(gearCatalog))
	setSizes := make(map[string]int)
	for _, gear := range gearCatalog {
		if strings.TrimSpace(gear.ID) == "" {
			return fmt.Errorf("gear %q has an empty ID", gear.Name)
		}
		if _, exists := seen[gear.ID]; exists {
			return fmt.Errorf("duplicate gear ID %q", gear.ID)
		}
		seen[gear.ID] = struct{}{}
		if setID := gear.EffectiveSetID(); setID != "" {
			setSizes[setID]++
		}
		for _, effect := range append([]ItemEffect{gear.Special}, gear.BonusEffects...) {
			if effect != EffectNone && ItemEffectDescription(effect) == "" {
				return fmt.Errorf("gear %q uses undocumented effect %q", gear.ID, effect)
			}
		}
	}
	for setID, size := range setSizes {
		if size < 2 {
			return fmt.Errorf("gear set %q has %d piece, want at least 2", setID, size)
		}
	}
	for _, effect := range effects {
		if strings.TrimSpace(ItemEffectDescription(effect)) == "" {
			return fmt.Errorf("item effect %q has no player-facing description", effect)
		}
	}
	return nil
}
