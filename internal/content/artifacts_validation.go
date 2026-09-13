package content

import (
	"fmt"
	"strings"
)

var itemEffectDescriptions = map[ItemEffect]string{
	EffectThorns:         "Reflects 10% of damage taken; duplicate copies diminish, capped at 40% (including shield boosts)",
	EffectVampiric:       "Heals for 5% of damage dealt; duplicate copies give 1/2, 1/3, … strength, capped at 30%",
	EffectBerserk:        "+20% damage while below 50% HP; duplicates diminish, capped at +40%",
	EffectLucky:          "+10% Luck; duplicate copies give 1/2, 1/3, … strength, capped at +40%",
	EffectTreasureHunter: "+5% item find chance; duplicate copies diminish, capped at +40%",
	EffectQuick:          "+10% Speed; duplicate copies give 1/2, 1/3, … strength, capped at +40%",
	EffectBulwark:        "+10% Defense; duplicate copies give 1/2, 1/3, … strength, capped at +40%",
	EffectRadiant:        "+10% XP gained; duplicate copies give 1/2, 1/3, … strength, capped at +40%",
	EffectFragile:        "+30% damage but double durability loss; duplicates diminish, capped at +40%",
	EffectSteady:         "-50% stun chance",
	EffectMindControl:    "Chance to capture low-health mobs",
	EffectRegenStack:     "Permanent regen stack on victory",
	EffectPhoenix:        "Revive once per fight at 50% HP",
	EffectStealth:        "Skip first-round mob damage",
	EffectParry:          "10% chance to negate a hit and counter",
	EffectCleanse:        "Removes a negative effect each turn",
	EffectExecutioner:    "+25% damage to targets below 30% HP; duplicate copies diminish, capped at +40%",
	EffectFocused:        "+10% Critical rating; duplicate copies give 1/2, 1/3, … strength, capped at +40%",
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
