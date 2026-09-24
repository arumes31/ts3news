package bot

import (
	"encoding/json"
	"testing"
	"ts3news/internal/content"
	"ts3news/internal/rift"
)

func TestRiftEquipmentSnapshotKeepsOriginalStats(t *testing.T) {
	u := UserInCombat{Equipped: map[content.GearSlot]content.Gear{content.SlotMainHand: {Name: "Saved sword", Slot: content.SlotMainHand, Stats: content.Stats{STR: 20, HP: -4}}}}
	build := riftBuildFromUser(u, "Test", 1)
	u.Equipped[content.SlotMainHand] = content.Gear{Name: "Replacement", Stats: content.Stats{STR: 99}}
	raw, err := json.Marshal(build)
	if err != nil {
		t.Fatal(err)
	}
	var saved rift.Build
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	gear := saved.Equipment[content.SlotMainHand]
	if gear.Name != "Saved sword" || gear.Stats.STR != 20 || gear.Stats.HP != -4 {
		t.Fatal("snapshot changed with account equipment")
	}
	empty := riftBuildFromUser(UserInCombat{}, "Test", 1)
	if empty.Equipment == nil || len(empty.Equipment) != 0 {
		t.Fatal("empty equipment must differ from missing old snapshot")
	}
}
