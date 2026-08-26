package bot

import "fmt"

type abyssPetAbility struct {
	Name       string
	Kind       string
	Cooldown   int
	PowerScale float64
}

func abyssPetAbilityForSlot(slot int) (abyssPetAbility, bool) {
	switch slot {
	case 1:
		return abyssPetAbility{Name: "Pounce", Kind: "attack", Cooldown: 3, PowerScale: 1.5}, true
	case 2:
		return abyssPetAbility{Name: "Healing Spell", Kind: "heal", Cooldown: 2, PowerScale: 0.15}, true
	default:
		return abyssPetAbility{}, false
	}
}

func abyssPetAbilityForClass(slot int, class string) (abyssPetAbility, bool) {
	if class == "support" && slot > 0 {
		return abyssPetAbility{Name: "Mending Cry", Kind: "heal", Cooldown: 2, PowerScale: 0.12}, true
	}
	return abyssPetAbilityForSlot(slot)
}

func abyssPetAbilityLabel(slot int) string {
	ability, active := abyssPetAbilityForSlot(slot)
	if !active {
		return "Reserve · assign an active slot to unlock an ability"
	}
	return fmt.Sprintf("%s · %d-round cooldown", ability.Name, ability.Cooldown)
}

func abyssPetAbilityLabelForClass(slot int, class string) string {
	ability, active := abyssPetAbilityForClass(slot, class)
	if !active {
		return "Reserve · assign an active slot to unlock an ability"
	}
	return fmt.Sprintf("%s · %d-round cooldown", ability.Name, ability.Cooldown)
}

func setAbyssPetAbilityCooldown(user *activeUser, petIndex, cooldown int) {
	if user.petCooldowns == nil {
		user.petCooldowns = map[int]int{}
	}
	user.petCooldowns[petIndex] = cooldown
}

func tickAbyssPetAbilityCooldowns(user *activeUser) []string {
	if user == nil || len(user.petCooldowns) == 0 {
		return nil
	}
	logs := make([]string, 0, len(user.petCooldowns))
	for petIndex, cooldown := range user.petCooldowns {
		if cooldown > 1 {
			user.petCooldowns[petIndex] = cooldown - 1
			continue
		}
		delete(user.petCooldowns, petIndex)
		if user.u == nil || petIndex < 0 || petIndex >= len(user.u.Pets) || user.u.Pets[petIndex] == nil {
			continue
		}
		ability, active := abyssPetAbilityForClass(petIndex+1, user.u.Pets[petIndex].PetClass)
		if active {
			logs = append(logs, fmt.Sprintf("🐾 %s's %s is ready.", user.u.Pets[petIndex].Name, ability.Name))
		}
	}
	return logs
}
