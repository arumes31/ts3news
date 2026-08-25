package bot

import "ts3news/internal/content"

const abyssRunFlagMimicsSurvived = "mimics_survived"

func advanceAbyssMimicChain(flags map[string]int64) bool {
	flags[abyssRunFlagMimicsSurvived]++
	return flags[abyssRunFlagMimicsSurvived] >= 3
}

func resetAbyssMimicChain(flags map[string]int64) {
	flags[abyssRunFlagMimicsSurvived] = 0
}

func abyssMimicKingSurvivalHP(currentHP, maxHP int) int {
	damage := max(1, maxHP/3)
	return max(1, currentHP-damage)
}

func abyssMimicKingGrant() (string, abyssLootGrant) {
	return "👑 Crown of False Gold [Unique]", abyssLootGrant{
		Type: "unique", UniqName: "Crown of False Gold", UniqRar: content.RarityMythic, UniqPow: 2.5,
	}
}
