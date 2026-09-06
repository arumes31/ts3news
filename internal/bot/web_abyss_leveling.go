package bot

import "math"

// Deeper victories pay a bounded depth bonus. Tier rewards grow by their square
// root so gold multipliers cannot turn into equally large leveling shortcuts.
// Defeats keep only the original consolation roll to discourage suicide farming.
func abyssCombatFloorXP(roll, depth int, tier abyssTier, victory bool) int {
	roll = min(max(roll, 1), 20)
	if !victory {
		return (roll + 3) / 4
	}
	base := roll + min(max(depth-1, 0), 400)/5
	return int(float64(base) * math.Sqrt(max(tier.RewardMult, 1)))
}
