package content

import "math"

// CriticalChance converts CRT rating into a percentage. Fifty rating is 25%
// (half of the 50% ceiling); at low ratings each point is nearly one percent.
// Unlike a hard clamp, additional rating always provides a smaller benefit.
func CriticalChance(rating int) float64 {
	return ratingChance(rating, 50)
}

// DodgeChance converts DGE rating into a percentage. Twenty-five rating is
// 12.5% (half of the 25% ceiling), with the same low-rating slope as CRT.
func DodgeChance(rating int) float64 {
	return ratingChance(rating, 25)
}

func ratingChance(rating int, ceiling float64) float64 {
	if rating <= 0 {
		return 0
	}
	return ceiling * (float64(rating) / (float64(rating) + ceiling))
}

// BonusEffectBudget bounds added affixes independently of the native Special.
// The two-affix floor preserves named legendary items and their imbue slot.
func BonusEffectBudget(rarity Rarity) int {
	switch rarity {
	case RarityDivine, RarityCelestial:
		return 3
	case RarityEternal:
		return 4
	default:
		return 2
	}
}

// AddedEffects removes duplicate, empty and native entries without discarding
// distinct legacy rolls above today's budget. Forge mutations normalize this
// list before counting capacity so rerolling Special cannot consume a phantom slot.
func (g Gear) AddedEffects() []ItemEffect {
	effects := make([]ItemEffect, 0, len(g.BonusEffects))
	have := map[ItemEffect]bool{EffectNone: true, g.Special: true}
	for _, effect := range g.BonusEffects {
		if !have[effect] {
			effects = append(effects, effect)
			have[effect] = true
		}
	}
	return effects
}

// Effects returns the effective, distinct affixes in stable order. Legacy items
// with too many added affixes retain their stored roll, but only the rarity's
// budget is active. Special never consumes that budget.
func (g Gear) Effects() []ItemEffect {
	effects := make([]ItemEffect, 0, 1+BonusEffectBudget(g.Rarity))
	if g.Special != EffectNone {
		effects = append(effects, g.Special)
	}
	added := g.AddedEffects()
	return append(effects, added[:min(len(added), BonusEffectBudget(g.Rarity))]...)
}

// PassiveEffectBonus aggregates identical passives using harmonic diminishing
// returns: first copy full strength, second half, third one third. Bonuses add
// once to the underlying stat, never compound through repeated multiplication.
func PassiveEffectBonus(effects []ItemEffect, effect ItemEffect) float64 {
	base, ceiling := 0.0, 0.4
	switch effect {
	case EffectThorns, EffectLucky, EffectQuick, EffectBulwark, EffectFocused, EffectRadiant:
		base = .1
	case EffectTreasureHunter:
		base = .05
	case EffectVampiric:
		base = .05
		ceiling = .3
	case EffectExecutioner:
		base = .25
	case EffectBerserk:
		base = .2
	case EffectFragile:
		base = .3
	default:
		return 0
	}
	count, bonus := 0, 0.0
	for _, candidate := range effects {
		if candidate != effect {
			continue
		}
		count++
		bonus += base / float64(count)
		if bonus >= ceiling {
			return ceiling
		}
	}
	return bonus
}

// ApplyPassiveStats is shared by combat totals and loadout previews.
func ApplyPassiveStats(stats Stats, effects []ItemEffect) Stats {
	stats.LCK = passiveStatValue(stats.LCK, PassiveEffectBonus(effects, EffectLucky))
	stats.SPD = passiveStatValue(stats.SPD, PassiveEffectBonus(effects, EffectQuick))
	stats.DEF = passiveStatValue(stats.DEF, PassiveEffectBonus(effects, EffectBulwark))
	stats.CRT = passiveStatValue(stats.CRT, PassiveEffectBonus(effects, EffectFocused))
	return stats
}

// Correct one ULP of multiplication noise before whole-stat truncation (for
// example 100 * 1.15 can otherwise become 114.99999999999999 and lose a point).
func passiveStatValue(value int, bonus float64) int {
	result := float64(value) * (1 + bonus)
	return int(math.Nextafter(result, math.Copysign(math.Inf(1), result)))
}
