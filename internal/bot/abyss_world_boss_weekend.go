package bot

import "time"

const abyssWorldBossWeekendMultiplier = 2

func abyssWorldBossWeekend(now time.Time) bool {
	weekday := now.UTC().Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

func abyssWorldBossStrikeMultiplier(now time.Time) int {
	if abyssWorldBossWeekend(now) {
		return abyssWorldBossWeekendMultiplier
	}
	return 1
}

func applyAbyssWorldBossWeekendReward(now time.Time, damage int64, drop abyssWeeklyBossDrop) (int64, abyssWeeklyBossDrop) {
	multiplier := abyssWorldBossStrikeMultiplier(now)
	drop.Amount *= multiplier
	return damage * int64(multiplier), drop
}
