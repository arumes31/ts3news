package main

import "math"

// XPForLevel returns the cumulative XP required to reach the given level.
// Mirrors the real bot's leveling.XPForLevel: Pow(level-1, 1.65) * 5.0
// This produces a much steeper curve than the old formula, preventing
// players from reaching level 1000+ in just 100 fights.
func XPForLevel(level int, exponentCap float64) float64 {
	if level <= 1 {
		return 0
	}
	// Match real bot: Pow(level-1, 1.65) * 5.0
	// At level 100: 99^1.65 * 5 ≈ 9,812 XP needed
	// At level 500: 499^1.65 * 5 ≈ 141,526 XP needed
	// At level 1000: 999^1.65 * 5 ≈ 530,000 XP needed
	base := 1.65
	exponent := math.Min(base, exponentCap)
	return math.Round(math.Pow(float64(level-1), exponent) * 5.0)
}

// XPNeededForNextLevel returns the XP delta between level and level+1.
func XPNeededForNextLevel(level int, exponentCap float64) float64 {
	return XPForLevel(level+1, exponentCap) - XPForLevel(level, exponentCap)
}

// LevelForXP returns the level that corresponds to the given total XP.
// Mirrors leveling.LevelForXP.
func LevelForXP(xp float64, exponentCap float64) int {
	if xp < 0 {
		return 1
	}
	lo, hi := 1, 1000000
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if XPForLevel(mid, exponentCap) <= xp {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}

// AwardXP adds XP to a player, handles level-ups and prestige.
// Returns the number of level-ups that occurred.
func AwardXP(p *SimPlayer, xp float64, params SimParams) int {
	if xp == 0 {
		return 0
	}

	p.XP += xp
	if xp > 0 {
		p.TotalXPEarned += xp
	}

	levelUps := 0
	for p.XP >= XPNeededForNextLevel(p.Level, params.ExponentCap) && p.Level < 10000 {
		p.XP -= XPNeededForNextLevel(p.Level, params.ExponentCap)
		p.Level++
		levelUps++
		p.RecalculateStats(params)

		// Prestige check
		if p.Level >= params.PrestigeLevel {
			p.Prestige++
			p.Level = 1
			p.XP = 0
			p.RecalculateStats(params)
		}
	}

	// Ensure XP doesn't go negative from penalties
	if p.XP < 0 {
		p.XP = 0
	}

	return levelUps
}

// ApplyDeathPenalty reduces XP on combat loss.
func ApplyDeathPenalty(p *SimPlayer, params SimParams) {
	penalty := p.XP * params.DeathXPPenalty
	if penalty < 10 {
		penalty = 10
	}
	p.XP -= penalty
	if p.XP < 0 {
		p.XP = 0
	}
}
