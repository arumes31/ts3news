package content

import (
	"math"
	"testing"
)

func TestEffectiveGearXPMatchesSlotEligibility(t *testing.T) {
	for _, test := range []struct {
		name string
		gear Gear
		want float64
	}{
		{"common is ineligible", Gear{Slot: SlotHead, Rarity: RarityCommon, XPMultiplier: 2}, 1},
		{"uncommon is ineligible", Gear{Slot: SlotHead, Rarity: RarityUncommon, XPMultiplier: 1.05}, 1},
		{"rare head keeps full bonus", Gear{Slot: SlotHead, Rarity: RarityRare, XPMultiplier: 1.3}, 1.3},
		{"neck bonus is capped", Gear{Slot: SlotNeck, Rarity: RarityEternal, XPMultiplier: 10}, 1.02},
		{"XP penalties remain penalties", Gear{Slot: SlotNeck, Rarity: RarityRare, XPMultiplier: 0.5}, 0.5},
		{"pet gear does not affect player XP", Gear{Slot: SlotPet1, Rarity: RarityEternal, XPMultiplier: 10}, 1},
		{"hidden gear is inert", Gear{Slot: SlotHead, Rarity: RarityRare, XPMultiplier: 10, Unidentified: true}, 1},
		{"negative is invalid", Gear{Slot: SlotHead, Rarity: RarityRare, XPMultiplier: -1}, 1},
		{"NaN is invalid", Gear{Slot: SlotHead, Rarity: RarityRare, XPMultiplier: math.NaN()}, 1},
		{"positive infinity is invalid even on capped slots", Gear{Slot: SlotNeck, Rarity: RarityRare, XPMultiplier: math.Inf(1)}, 1},
		{"negative infinity is invalid", Gear{Slot: SlotHead, Rarity: RarityRare, XPMultiplier: math.Inf(-1)}, 1},
		{"zero remains a valid penalty", Gear{Slot: SlotHead, Rarity: RarityRare, XPMultiplier: 0}, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.gear.EffectiveXPMultiplier(); got != test.want {
				t.Fatalf("effective XP = %v, want %v", got, test.want)
			}
		})
	}
}
