package bot

import (
	"fmt"

	"ts3news/internal/content"
)

func lootSyncGearGroupName(
	gear content.Gear,
	slot string,
	formatGSName func(score int, name string, effect content.ItemEffect, itemType string) string,
) (string, bool) {
	if !abyssGearActiveForCombat(gear) {
		return "", false
	}
	if content.IsPetGearSlot(content.GearSlot(slot)) {
		return "", false
	}
	name := gear.Name
	if gear.GearLevel > 0 {
		name = fmt.Sprintf("%s +%d", name, gear.GearLevel)
	}
	return formatGSName(gear.Stats.Score(), name, gear.Special, "slot:"+slot), true
}
