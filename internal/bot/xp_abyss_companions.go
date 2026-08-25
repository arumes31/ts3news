package bot

import "ts3news/internal/content"

func abyssCapturedPetLoyalty(currentHP, maxHP int) int {
	if maxHP <= 0 {
		return 1
	}
	return min(100, max(1, currentHP*100/maxHP))
}

func abyssPetNervous(loyalty int) bool {
	return loyalty > 0 && loyalty < 10
}

func abyssMindControlCapture(target *content.Mob) {
	if target == nil {
		return
	}
	target.Loyalty = abyssCapturedPetLoyalty(target.Stats.HP, target.MaxHP)
	target.Stats.HP = 1
	target.CurrentHP = 1
}
