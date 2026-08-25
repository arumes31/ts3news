package bot

func abyssPostHocSurvivalChance(depth int, tier abyssTier, playerCR float64) int {
	return max(0, min(100, 100-abyssRiskPct(depth, tier, playerCR)))
}
