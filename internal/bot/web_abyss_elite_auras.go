package bot

import (
	"fmt"

	"ts3news/internal/content"
)

type abyssEliteAura struct {
	Name   string
	Stat   string
	Effect content.MobEffect
}

func abyssEliteAuraForDepth(depth int) (abyssEliteAura, bool) {
	if depth <= 0 || depth%3 != 0 {
		return abyssEliteAura{}, false
	}
	auras := []abyssEliteAura{
		{Name: "Blood Chorus", Stat: "STR", Effect: content.EffectEnraged},
		{Name: "Iron Canticle", Stat: "DEF", Effect: content.EffectArmored},
		{Name: "Gale Hymn", Stat: "SPD", Effect: content.EffectFleet},
	}
	return auras[(depth/3-1)%len(auras)], true
}

func applyScheduledAbyssEliteAura(depth int, mobs []content.Mob) ([]content.Mob, string) {
	aura, active := abyssEliteAuraForDepth(depth)
	if !active || len(mobs) == 0 {
		return mobs, ""
	}

	carrier := -1
	for i := range mobs {
		switch mobs[i].Type {
		case content.MobElite, content.MobMiniboss, content.MobBoss, content.MobLegendary:
			carrier = i
		}
		if carrier >= 0 {
			break
		}
	}
	if carrier < 0 {
		for i := range mobs {
			if abyssEnemyHazard(&mobs[i]) || mobs[i].Type == content.MobTreasureGoblin {
				continue
			}
			if carrier < 0 || mobs[i].Stats.HP > mobs[carrier].Stats.HP {
				carrier = i
			}
		}
	}
	if carrier < 0 {
		return mobs, ""
	}
	if mobs[carrier].Type == content.MobCommon || mobs[carrier].Type == content.MobEliteMinion {
		mobs[carrier].Type = content.MobElite
	}
	if !mobHasEffect(mobs[carrier], aura.Effect) {
		mobs[carrier].Effects = append(mobs[carrier].Effects, aura.Effect)
	}

	for i := range mobs {
		switch aura.Stat {
		case "STR":
			mobs[i].Stats.STR = max(1, mobs[i].Stats.STR*110/100)
		case "DEF":
			mobs[i].Stats.DEF = max(1, mobs[i].Stats.DEF*110/100)
		case "SPD":
			mobs[i].Stats.SPD = max(1, mobs[i].Stats.SPD*110/100)
		}
	}
	return mobs, fmt.Sprintf("🔱 Elite aura — %s: %s empowers the enemy pack with +10%% %s.",
		aura.Name, mobs[carrier].Name, aura.Stat)
}
