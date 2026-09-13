package bot

import "ts3news/internal/content"

func abyssGearActiveForCombat(gear content.Gear) bool {
	return !gear.Unidentified
}

func abyssPlayerEquipment(equipped map[content.GearSlot]content.Gear) map[content.GearSlot]content.Gear {
	player := make(map[content.GearSlot]content.Gear, len(equipped))
	for slot, gear := range equipped {
		if !content.IsPetGearSlot(slot) && abyssGearActiveForCombat(gear) {
			player[slot] = gear
		}
	}
	return player
}

func abyssPetGearStats(equipped map[content.GearSlot]content.Gear) content.Stats {
	var stats content.Stats
	for slot, gear := range equipped {
		if content.IsPetGearSlot(slot) && abyssGearActiveForCombat(gear) {
			stats = stats.Add(gear.Stats)
		}
	}
	return stats
}

func applyAbyssPetGear(pet *content.Mob, bonus content.Stats) {
	if pet == nil {
		return
	}
	currentHP := pet.Stats.HP
	pet.Stats = pet.Stats.Add(bonus)
	pet.MaxHP = max(1, pet.MaxHP+bonus.HP)
	pet.Stats.HP = max(0, min(currentHP, pet.MaxHP))
}

// healAbyssPet returns only health restored, never the requested overheal.
func healAbyssPet(pet *content.Mob, amount int) int {
	if pet == nil || pet.Stats.HP <= 0 {
		return 0
	}
	pet.MaxHP = max(1, pet.MaxHP)
	before := min(pet.Stats.HP, pet.MaxHP)
	restored := min(max(0, amount), pet.MaxHP-before)
	pet.Stats.HP = before + restored
	pet.CurrentHP = pet.Stats.HP
	return restored
}

func restoreAbyssPetHealth(pet *content.Mob, baseHP, baseMaxHP int, health *abyssPetHealthState) {
	if pet == nil {
		return
	}
	hp := min(max(0, baseHP), max(1, baseMaxHP))
	if health != nil && health.BaseHP == baseHP && health.BaseMaxHP == baseMaxHP {
		hp = health.HP
	}
	pet.MaxHP = max(1, pet.MaxHP)
	pet.Stats.HP = min(max(0, hp), pet.MaxHP)
	pet.CurrentHP = pet.Stats.HP
}
