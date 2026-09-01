package bot

import (
	"fmt"
	"testing"

	"ts3news/internal/content"
)

func TestLootSyncGearGroupOmitsUnidentifiedGear(t *testing.T) {
	hidden := content.Gear{
		Name: "Crown of the Last Star", Stats: content.Stats{STR: 9_999},
		Special: content.EffectExecutioner, GearLevel: 12, Unidentified: true,
	}
	formatterCalled := false
	groupName, ok := lootSyncGearGroupName(hidden, "Head", func(score int, name string, effect content.ItemEffect, itemType string) string {
		formatterCalled = true
		return fmt.Sprintf("%d|%s|%s|%s", score, name, effect, itemType)
	})
	if ok || formatterCalled || groupName != "" {
		t.Fatalf("unidentified gear produced group %q (ok=%t, formatter called=%t)", groupName, ok, formatterCalled)
	}
}

func TestLootSyncGearGroupOmitsPetGear(t *testing.T) {
	petGear := content.Gear{
		Name: "Companion Harness", Slot: content.SlotPet1, Stats: content.Stats{DEF: 12},
	}
	formatterCalled := false
	groupName, ok := lootSyncGearGroupName(petGear, string(content.SlotPet1), func(score int, name string, effect content.ItemEffect, itemType string) string {
		formatterCalled = true
		return fmt.Sprintf("%d|%s|%s|%s", score, name, effect, itemType)
	})
	if ok || formatterCalled || groupName != "" {
		t.Fatalf("pet gear produced group %q (ok=%t, formatter called=%t)", groupName, ok, formatterCalled)
	}
}

func TestLootSyncGearGroupUsesIdentifiedInstance(t *testing.T) {
	gear := content.Gear{
		Name: "Iron Crown", Stats: content.Stats{STR: 12},
		Special: content.EffectThorns, GearLevel: 3,
	}
	groupName, ok := lootSyncGearGroupName(gear, "Head", func(score int, name string, effect content.ItemEffect, itemType string) string {
		return fmt.Sprintf("%d|%s|%s|%s", score, name, effect, itemType)
	})
	want := fmt.Sprintf("%d|Iron Crown +3|%s|slot:Head", gear.Stats.Score(), content.EffectThorns)
	if !ok || groupName != want {
		t.Fatalf("identified gear group = %q (ok=%t), want %q", groupName, ok, want)
	}
}
