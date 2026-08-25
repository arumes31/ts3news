package bot

import (
	"fmt"

	"ts3news/internal/content"
)

type abyssNamedRareDefinition struct {
	Name      string
	Signature string
	Element   content.Element
	Power     float64
}

var abyssNamedRareDefinitions = []abyssNamedRareDefinition{
	{Name: "Sableclaw, the Unmoored", Signature: "Sableclaw's Phase Fang", Element: content.ElementAir, Power: 8},
	{Name: "Ilyra, Bell of Ash", Signature: "Ilyra's Cinder Chime", Element: content.ElementFire, Power: 9},
	{Name: "Orryx, the Drowned Star", Signature: "Orryx's Tidal Lens", Element: content.ElementWater, Power: 10},
	{Name: "Vael, Keeper of Moths", Signature: "Vael's Moonwing Mantle", Element: content.ElementEarth, Power: 11},
}

func abyssNamedRareSpawn(depth int, random combatRandomSource) (content.Mob, string, bool) {
	if depth < 6 || random.Float64() >= 0.04 {
		return content.Mob{}, "", false
	}
	rare := abyssNamedRareDefinitions[random.IntN(len(abyssNamedRareDefinitions))]
	hp := max(240, depth*30)
	return content.Mob{
		Name:      rare.Name,
		Type:      content.MobElite,
		Level:     depth + 3,
		Stats:     content.Stats{HP: hp, STR: max(20, depth*3), DEF: max(10, depth), SPD: 35},
		CurrentHP: hp,
		MaxHP:     hp,
		Element:   rare.Element,
		RewardXP:  depth * 8,
	}, rare.Signature, true
}

func abyssNamedRareDrop(name string) (string, abyssLootGrant, bool) {
	for _, rare := range abyssNamedRareDefinitions {
		if rare.Name == name {
			label := fmt.Sprintf("💎 Signature Relic: %s [Epic]", rare.Signature)
			return label, abyssLootGrant{
				Type: "unique", UniqName: rare.Signature, UniqRar: content.RarityEpic, UniqPow: rare.Power,
			}, true
		}
	}
	return "", abyssLootGrant{}, false
}
