package bot

import (
	"testing"

	"ts3news/internal/content"
)

func TestDecideGearFate(t *testing.T) {
	// A weak common item and a strong rare item in the same slot.
	weakCommon := content.Gear{
		Name: "Rusty Helm", Slot: content.SlotHead, Rarity: content.RarityCommon,
		Stats: content.Stats{HP: 10, STR: 1, DEF: 1},
	}
	strongRare := content.Gear{
		Name: "Dragon Helm", Slot: content.SlotHead, Rarity: content.RarityRare,
		Stats: content.Stats{HP: 200, STR: 40, DEF: 30, CRT: 10},
	}
	strongEpic := content.Gear{
		Name: "Void Crown", Slot: content.SlotHead, Rarity: content.RarityEpic,
		Stats: content.Stats{HP: 300, STR: 60, DEF: 40, CRT: 15},
	}

	tests := []struct {
		name    string
		drop    content.Gear
		current *content.Gear
		want    lootFate
	}{
		{"empty slot is always equipped", strongRare, nil, fateEquip},
		{"upgrade is equipped", strongEpic, &strongRare, fateEquip},
		{"rare non-upgrade goes to auction house", strongRare, &strongEpic, fateListAH},
		{"common non-upgrade is salvaged", weakCommon, &strongRare, fateSalvage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decideGearFate(tt.drop, tt.current); got != tt.want {
				t.Errorf("decideGearFate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDecideGearFate_RarePlusNeverScrapped is the regression guard for the bug:
// a rare-or-better drop must never be salvaged into scrap — it is always either
// equipped or listed on the auction house.
func TestDecideGearFate_RarePlusNeverScrapped(t *testing.T) {
	// Current gear is deliberately overpowered so no drop can be an upgrade,
	// forcing the rarity gate to decide between AH and scrap.
	current := content.Gear{
		Slot: content.SlotChest, Rarity: content.RarityDivine,
		Stats: content.Stats{HP: 100000, STR: 9000, DEF: 9000, CRT: 100},
	}

	for rar := content.RarityCommon; rar <= content.RarityMythic; rar++ {
		drop := content.Gear{
			Slot: content.SlotChest, Rarity: rar,
			Stats: content.Stats{HP: 10, STR: 1, DEF: 1},
		}
		fate := decideGearFate(drop, &current)
		if rar >= content.RarityRare && fate == fateSalvage {
			t.Errorf("rarity %v was salvaged into scrap; rare+ must be actioned", rar)
		}
		if rar < content.RarityRare && fate != fateSalvage {
			t.Errorf("rarity %v non-upgrade should salvage, got %v", rar, fate)
		}
	}
}

func TestScrapValue(t *testing.T) {
	low := content.Gear{Rarity: content.RarityCommon, Stats: content.Stats{HP: 10}}
	high := content.Gear{Rarity: content.RarityEpic, Stats: content.Stats{HP: 300, STR: 60}}

	if scrapValue(low) < 1 {
		t.Errorf("scrapValue should be at least 1, got %d", scrapValue(low))
	}
	if scrapValue(high) <= scrapValue(low) {
		t.Errorf("higher rarity/stats should yield more scrap: high=%d low=%d",
			scrapValue(high), scrapValue(low))
	}
}
