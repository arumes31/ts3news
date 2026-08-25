package bot

func combatBossEnrageRound(isAbyss bool) int {
	if isAbyss {
		return 30
	}
	return 8
}

func combatBossShouldEnrage(round int, isAbyss bool) bool {
	return round >= combatBossEnrageRound(isAbyss)
}
