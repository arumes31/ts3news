package bot

import "ts3news/internal/content"

func abyssPlayerEquipment(equipped map[content.GearSlot]content.Gear) map[content.GearSlot]content.Gear {
	player := make(map[content.GearSlot]content.Gear, len(equipped))
	for slot, gear := range equipped {
		if !content.IsPetGearSlot(slot) {
			player[slot] = gear
		}
	}
	return player
}

func abyssPetGearStats(equipped map[content.GearSlot]content.Gear) content.Stats {
	var stats content.Stats
	for slot, gear := range equipped {
		if content.IsPetGearSlot(slot) {
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
