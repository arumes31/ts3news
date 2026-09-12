package bot

import (
	"math"

	"ts3news/internal/content"
)

func forgeStatIncrement(value, percent int) int { return max(1, value*percent/100) }

func forgePrismaticGain(stats content.Stats) (string, int) {
	code, value := "STR", stats.STR
	for _, candidate := range []string{"HP", "DEF", "SPD", "LCK", "INT", "STA", "CRT", "DGE", "MNA"} {
		if v := *gearStatRef(&stats, candidate); v > value {
			code, value = candidate, v
		}
	}
	return code, forgeStatIncrement(value, 5)
}

// forgeStatUpgradeResult is shared by previews and the corresponding commit
// paths. Costs, eligibility, random affixes and persistence remain with callers.
func forgeStatUpgradeResult(g content.Gear, operation string) content.Gear {
	switch operation {
	case "reinforce":
		g.Stats.DEF += forgeStatIncrement(g.Stats.DEF, 2)
	case "sharpen":
		g.Stats.STR += forgeStatIncrement(g.Stats.STR, 2)
	case "prismatic_rune":
		code, inc := forgePrismaticGain(g.Stats)
		*gearStatRef(&g.Stats, code) += inc
	case "temper":
		g.Stats = g.Stats.Scaled(1.02)
	case "masterwork":
		g.Stats = g.Stats.Scaled(1.03)
	case "attune":
		g.Stats = g.Stats.Scaled(1.05)
	case "infuse_curse", "infuse_eldritch":
		g.Stats = g.Stats.Scaled(1.25)
	case "upgrade_gear":
		g.Rarity++
		g.Stats = g.Stats.Scaled(1.3)
	}
	return g
}

func exactForgeStatOutcome(operation string, before content.Gear, chance float64) (abyssForgeOutcome, bool) {
	switch operation {
	case "reinforce", "sharpen", "prismatic_rune", "temper", "masterwork", "attune", "upgrade_gear", "infuse_curse", "infuse_eldritch":
	default:
		return abyssForgeOutcome{}, false
	}
	if math.IsNaN(chance) {
		chance = 0
	}
	chance = min(1, max(0, chance))
	if operation != "temper" {
		chance = 1
	}
	after := forgeStatUpgradeResult(before, operation)
	out := abyssForgeOutcome{MinimumStats: before.Stats, MaximumStats: before.Stats, ExpectedStats: before.Stats,
		MinimumCR: min(before.CombatRating(), after.CombatRating()), MaximumCR: max(before.CombatRating(), after.CombatRating()),
		ExpectedCR: (1-chance)*before.CombatRating() + chance*after.CombatRating()}
	if chance == 1 {
		out.MinimumStats, out.MaximumStats, out.ExpectedStats = after.Stats, after.Stats, after.Stats
		out.MinimumCR, out.MaximumCR, out.ExpectedCR = after.CombatRating(), after.CombatRating(), after.CombatRating()
	} else if chance == 0 {
		out.MinimumCR, out.MaximumCR, out.ExpectedCR = before.CombatRating(), before.CombatRating(), before.CombatRating()
	} else {
		for _, stat := range before.Stats.Details() {
			if !stat.Combat {
				continue
			}
			a, b := stat.Value, *gearStatRef(&after.Stats, stat.Code)
			*gearStatRef(&out.MinimumStats, stat.Code) = min(a, b)
			*gearStatRef(&out.MaximumStats, stat.Code) = max(a, b)
			*gearStatRef(&out.ExpectedStats, stat.Code) = int(float64(a)*(1-chance) + float64(b)*chance)
		}
	}
	if operation == "upgrade_gear" {
		out.TargetRarity = after.Rarity.String()
	}
	out.Gained = []string{"Preview uses the operation's actual whole-stat rounding."}
	if operation == "temper" {
		out.Consequences = []string{"Failure keeps the item's stats unchanged. Expected stats are probability-weighted outcomes, rounded toward zero; they are not a guaranteed roll."}
	}
	return out, true
}
