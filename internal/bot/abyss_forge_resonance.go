package bot

import (
	"fmt"
	"sort"
	"strings"

	"ts3news/internal/content"
)

// abyssGemResonanceBonus returns the live bonus granted when at least three
// equipped gems share a base family. The bonus is five percent of that
// family's socketed stat contribution, preserving each gem's tier scaling.
func abyssGemResonanceBonus(equipped map[content.GearSlot]content.Gear) (content.Stats, map[string]int) {
	counts := content.GemResonance(equipped)
	contributions := make(map[string]content.Stats)
	for _, gear := range equipped {
		for _, gem := range gear.Gemstones {
			base, tier := parseGem(gem)
			baseStats, valid := gemBaseStats(base)
			if !valid {
				continue
			}
			multiplier := []float64{0, 1, 2, 4}[min(3, max(0, tier))]
			contributions[base] = contributions[base].Add(baseStats.Scaled(multiplier))
		}
	}

	var bonus content.Stats
	for family, count := range counts {
		if count >= 3 {
			bonus = bonus.Add(contributions[family].Scaled(0.05))
		}
	}
	return bonus, counts
}

func forgeGemResonancePreview(equipped map[content.GearSlot]content.Gear, gear content.Gear, gem string) string {
	base, _, _ := strings.Cut(strings.TrimSpace(gem), " ")
	if _, valid := gemBaseStats(base); !valid {
		return ""
	}
	_, before := abyssGemResonanceBonus(equipped)
	projected := make(map[content.GearSlot]content.Gear, len(equipped)+1)
	for slot, item := range equipped {
		projected[slot] = item
	}
	gear.Gemstones = append(append([]string(nil), gear.Gemstones...), base)
	projected[gear.Slot] = gear
	bonus, after := abyssGemResonanceBonus(projected)
	state := "inactive"
	if after[base] >= 3 {
		state = "active: +5% of the socketed family's stat contribution"
	}
	return fmt.Sprintf("%s resonance %d→%d (%s; projected bonus %s)",
		base, before[base], after[base], state, compactForgeStats(bonus))
}

func forgeSetResonancePreview(equipped map[content.GearSlot]content.Gear, gear content.Gear) string {
	setID := gear.EffectiveSetID()
	if setID == "" {
		return ""
	}
	beforeCounts := forgeSetCounts(equipped)
	_, beforeTiers := content.AbyssSetBonusBySet(beforeCounts)
	projected := make(map[content.GearSlot]content.Gear, len(equipped)+1)
	for slot, item := range equipped {
		projected[slot] = item
	}
	projected[gear.Slot] = gear
	afterCounts := forgeSetCounts(projected)
	bonus, afterTiers := content.AbyssSetBonusBySet(afterCounts)
	return fmt.Sprintf("%s set resonance %d→%d pieces, active tier %d→%d (projected set bonus %s)",
		setID, beforeCounts[setID], afterCounts[setID], beforeTiers[setID], afterTiers[setID], compactForgeStats(bonus))
}

func forgeSetCounts(equipped map[content.GearSlot]content.Gear) map[string]int {
	counts := make(map[string]int)
	for _, gear := range equipped {
		if setID := gear.EffectiveSetID(); setID != "" {
			counts[setID]++
		}
	}
	return counts
}

func forgeRuneImpact(family, rune string) string {
	element := content.Element(strings.TrimSpace(rune))
	switch strings.ToLower(strings.TrimSpace(family)) {
	case "defensive":
		return fmt.Sprintf("%s Ward grants %d%% resistance against %s damage.", element, content.DefensiveRuneResistPct, element)
	default:
		if element == content.ElementPhysical {
			return "Physical rune grants +5% resonance to matching Physical attacks and deals neutral 1.0× damage against every element."
		}
		var effective, weak []string
		for _, defender := range []content.Element{content.ElementFire, content.ElementWater, content.ElementEarth, content.ElementAir} {
			switch getElementMult(element, defender) {
			case 2:
				effective = append(effective, string(defender))
			case 0.5:
				weak = append(weak, string(defender))
			}
		}
		return fmt.Sprintf("%s rune grants +5%% resonance to matching %s attacks; it deals 2.0× against %s, 0.5× against %s, and 1.0× otherwise.",
			element, element, strings.Join(effective, ", "), strings.Join(weak, ", "))
	}
}

func compactForgeStats(stats content.Stats) string {
	values := map[string]int{
		"HP": stats.HP, "STR": stats.STR, "DEF": stats.DEF, "SPD": stats.SPD,
		"LCK": stats.LCK, "INT": stats.INT, "STA": stats.STA, "CRT": stats.CRT,
		"DGE": stats.DGE, "MNA": stats.MNA,
	}
	keys := make([]string, 0, len(values))
	for key, value := range values {
		if value != 0 {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return "none"
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s %+d", key, values[key]))
	}
	return strings.Join(parts, ", ")
}
