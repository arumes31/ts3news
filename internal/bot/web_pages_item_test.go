package bot

import (
	"encoding/json"
	"testing"

	"ts3news/internal/content"
)

func TestGearAtlasViewUsesSemanticSlotRegions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		slot   content.GearSlot
		family string
	}{
		{name: "weapon", slot: content.SlotMainHand, family: "items"},
		{name: "helmet", slot: content.SlotHead, family: "items"},
		{name: "chest", slot: content.SlotChest, family: "items"},
		{name: "gloves", slot: content.SlotHands, family: "items"},
		{name: "boots", slot: content.SlotFeet, family: "items"},
		{name: "belt", slot: content.SlotWaist, family: "items"},
		{name: "ring", slot: content.SlotFinger1, family: "items"},
		{name: "necklace", slot: content.SlotNeck, family: "items"},
		{name: "ranged", slot: content.SlotRanged, family: "ranged"},
		{name: "artifact", slot: content.SlotArtifact, family: "artifacts"},
		{name: "banner", slot: content.SlotBanner, family: "banners"},
	}
	gearCatalog := content.GearAppearanceCatalog()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var gear content.Gear
			for _, candidate := range gearCatalog {
				if candidate.Slot == test.slot {
					gear = candidate
					break
				}
			}
			if gear.ID == "" {
				t.Fatalf("no catalog gear for slot %s", test.slot)
			}
			first := gearAtlasView(gear)
			second := gearAtlasView(gear)
			if first != second {
				t.Fatalf("atlas mapping is unstable: first=%#v second=%#v", first, second)
			}
			if first.Family != test.family {
				t.Errorf("family = %q, want %q", first.Family, test.family)
			}
			if first.Asset == "" {
				t.Error("exact atlas asset is empty")
			}
			if first.Column < 0 || first.Column >= content.PixelArtColumns || first.Row < 0 || first.Row >= content.PixelArtRows {
				t.Errorf("row = %d, want shared 14x12 grid coordinate", first.Row)
			}
		})
	}
}

func TestGearViewInspectorIncludesEverySpecial(t *testing.T) {
	t.Parallel()

	gear := content.Gear{
		ID:      "inspect-specials",
		Name:    "Special Inspector",
		Slot:    content.SlotMainHand,
		Rarity:  content.RarityMythic,
		Stats:   content.Stats{STR: 100, CRT: 20},
		Special: content.EffectVampiric,
		BonusEffects: []content.ItemEffect{
			content.EffectBerserk,
			content.EffectQuick,
			content.EffectBerserk,
		},
	}
	view := toGearView(gear.Slot, gear)
	var inspection itemInspectView
	if err := json.Unmarshal([]byte(view.InspectJSON), &inspection); err != nil {
		t.Fatalf("decode inspection JSON: %v", err)
	}
	if len(inspection.Specials) != 3 {
		t.Fatalf("specials = %#v, want base plus two unique bonus specials", inspection.Specials)
	}
	for _, special := range inspection.Specials {
		if special.Description == "" {
			t.Errorf("special %q has no description", special.Name)
		}
	}
}
