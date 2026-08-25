package bot

import "ts3news/internal/content"

const abyssDoubleBossDepth = 60

var abyssBossRoster = []string{
	"Gorgoroth the Firelord",
	"Malakor the Voidweaver",
	"Azazoth the Slumbering Eye",
}

func abyssBossNameAtDepth(depth int) string {
	switch {
	case depth == 100:
		return "Abyssus, Heart of the Void"
	case depth%20 == 5:
		return abyssBossRoster[0]
	case depth%20 == 10:
		return abyssBossRoster[1]
	case depth%20 == 15:
		return abyssBossRoster[2]
	default:
		return abyssBossRoster[(depth/5)%len(abyssBossRoster)]
	}
}

func abyssBossNamesAtDepth(depth int) []string {
	primary := abyssBossNameAtDepth(depth)
	if depth <= abyssDoubleBossDepth {
		return []string{primary}
	}
	partner := abyssBossRoster[(depth/5+1)%len(abyssBossRoster)]
	if partner == primary {
		partner = abyssBossRoster[(depth/5+2)%len(abyssBossRoster)]
	}
	return []string{primary, partner}
}

func abyssBossEncounter(depth, mobLevel int, difficulty float64) []content.Mob {
	names := abyssBossNamesAtDepth(depth)
	lvlScale, effectiveDiff := abyssMobScalars(mobLevel, difficulty)
	bossDef := min(10+mobLevel/2, 90)
	hpScale, damageScale, rewardXP := 1.0, 1.0, 500
	if len(names) > 1 {
		// Two full solo stat blocks would double the difficulty. Twin tyrants each
		// carry 70% health and 75% damage: a material 40–50% encounter escalation.
		hpScale, damageScale, rewardXP = 0.70, 0.75, 350
	}
	mobs := make([]content.Mob, 0, len(names))
	for _, name := range names {
		mobs = append(mobs, content.Mob{
			Name:  name,
			Type:  content.MobBoss,
			Level: mobLevel + 1,
			Stats: content.Stats{
				HP:  int(1000 * lvlScale * effectiveDiff * hpScale),
				STR: int(50 * lvlScale * abyssMobDamageMult * effectiveDiff * damageScale),
				DEF: bossDef,
				SPD: 105,
			},
			RewardXP: rewardXP,
			Element:  content.ElementPhysical,
		})
	}
	return mobs
}
