package rift

import (
	"encoding/json"
	"testing"
	"ts3news/internal/content"
)

func TestLootDescriptionPreservesOriginalLoreAndEffectFallback(t *testing.T) {
	for _, lore := range []string{"Original <Abyss> lore", ""} {
		gear := content.AbyssGearCatalog()[0]
		gear.Lore = lore
		raw, err := json.Marshal(Drop{ID: "test", Gear: &gear})
		if err != nil {
			t.Fatal(err)
		}
		var value struct {
			Description string `json:"gear_description"`
		}
		if err = json.Unmarshal(raw, &value); err != nil {
			t.Fatal(err)
		}
		want := lore
		if want == "" {
			want = content.ItemEffectDescription(gear.Special)
		}
		if value.Description != want {
			t.Fatal("description altered")
		}
	}
}
