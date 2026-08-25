package bot

import (
	"time"

	"ts3news/internal/content"
)

// corruptedConsumableBacklash returns the immediate self-damage caused by a
// corrupted consumable. Keeping the percentages here makes the live and lobby
// action paths share the same authoritative rule.
func corruptedConsumableBacklash(consumableID string, maxHP int) int {
	if maxHP <= 0 {
		return 0
	}
	percent := 0
	switch consumableID {
	case "corrupted_great_health_potion", "corrupted_rejuvenation_potion":
		percent = 10
	case "corrupted_strength_elixir":
		percent = 5
	}
	if percent == 0 {
		return 0
	}
	return max(1, maxHP*percent/100)
}

// sentimentalValueBonus grants one percent of an item's positive stats after
// it has remained in the collection for 30 days. Integer stats round down;
// sufficiently small attributes intentionally receive no artificial +1.
func sentimentalValueBonus(gear content.Gear, now time.Time) content.Stats {
	if !gear.BrokenIn(now) {
		return content.Stats{}
	}
	percent := func(value int) int {
		if value <= 0 {
			return 0
		}
		return value / 100
	}
	stats := gear.Stats
	return content.Stats{
		HP: percent(stats.HP), STR: percent(stats.STR), DEF: percent(stats.DEF),
		SPD: percent(stats.SPD), LCK: percent(stats.LCK), INT: percent(stats.INT),
		STA: percent(stats.STA), CRT: percent(stats.CRT), DGE: percent(stats.DGE),
		MNA: percent(stats.MNA), CHA: percent(stats.CHA), STN: percent(stats.STN),
		SHN: percent(stats.SHN), HGR: percent(stats.HGR),
	}
}

// defensiveRuneResistPct stacks matching armor wards while capping the total
// reduction so a heavily socketed loadout can never become damage immune.
func defensiveRuneResistPct(equipped map[content.GearSlot]content.Gear, element content.Element) int {
	if element == "" {
		return 0
	}
	resist := 0
	for _, gear := range equipped {
		wardElement, ok := content.ParseDefensiveRune(gear.Rune)
		if ok && wardElement == element {
			resist += content.DefensiveRuneResistPct
		}
	}
	return min(50, resist)
}
