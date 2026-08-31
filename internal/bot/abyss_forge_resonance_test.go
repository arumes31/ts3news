package bot

import (
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssGemResonanceBonusUsesTierContribution(t *testing.T) {
	equipped := map[content.GearSlot]content.Gear{
		content.SlotHead:     {Slot: content.SlotHead, Gemstones: []string{"Ruby"}},
		content.SlotChest:    {Slot: content.SlotChest, Gemstones: []string{"Ruby II"}},
		content.SlotMainHand: {Slot: content.SlotMainHand, Gemstones: []string{"Ruby III"}},
		content.SlotLegs:     {Slot: content.SlotLegs, Gemstones: []string{"Ruby III"}, Unidentified: true},
		content.SlotPet1:     {Slot: content.SlotPet1, Gemstones: []string{"Ruby III"}},
	}
	bonus, counts := abyssGemResonanceBonus(equipped)
	if counts["Ruby"] != 3 {
		t.Fatalf("Ruby resonance count = %d, want 3", counts["Ruby"])
	}
	// Ruby contributes 100 HP at tier I, 200 at II, and 400 at III. Five
	// percent of 700 rounds to 35 with Stats.Scaled.
	if bonus.HP != 35 {
		t.Fatalf("Ruby resonance HP = %d, want 35", bonus.HP)
	}
}

func TestForgeSetCountsIgnoresUnidentifiedAndPetGear(t *testing.T) {
	equipped := map[content.GearSlot]content.Gear{
		content.SlotHead:  {Slot: content.SlotHead, SetID: "predator"},
		content.SlotChest: {Slot: content.SlotChest, SetID: "predator", Unidentified: true},
		content.SlotPet1:  {Slot: content.SlotPet1, SetID: "predator"},
	}
	if counts := forgeSetCounts(equipped); counts["predator"] != 1 {
		t.Fatalf("forge set counts = %#v, want one active predator piece", counts)
	}
}

func TestForgeRuneImpactPublishesExactMatchups(t *testing.T) {
	offense := forgeRuneImpact("offensive", "Fire")
	if !strings.Contains(offense, "+5% resonance to matching Fire attacks") ||
		!strings.Contains(offense, "2.0× against Air") || !strings.Contains(offense, "0.5× against Water") {
		t.Fatalf("offensive impact = %q", offense)
	}
	defense := forgeRuneImpact("defensive", "Water")
	if !strings.Contains(defense, "5% resistance") {
		t.Fatalf("defensive impact = %q", defense)
	}
}

func TestForgeCorruptionOutcomeShowsProtectionDistribution(t *testing.T) {
	gear := content.Gear{Stats: content.Stats{HP: 1000, STR: 100, DEF: 80, SPD: 60}}
	plain := forgeCorruptionOutcome(gear, false)
	protected := forgeCorruptionOutcome(gear, true)
	if plain.MaximumStats.HP != gear.Stats.HP || plain.MinimumStats.HP >= plain.MaximumStats.HP {
		t.Fatalf("plain corruption range = %+v to %+v", plain.MinimumStats, plain.MaximumStats)
	}
	plainLoss := gear.Stats.HP - plain.MinimumStats.HP
	protectedLoss := gear.Stats.HP - protected.MinimumStats.HP
	if protectedLoss != plainLoss/2 {
		t.Fatalf("protected HP loss = %d, want half of %d", protectedLoss, plainLoss)
	}
	if protected.ExpectedStats.HP <= protected.MinimumStats.HP || protected.ExpectedStats.HP >= protected.MaximumStats.HP {
		t.Fatalf("expected HP %d is outside distribution", protected.ExpectedStats.HP)
	}
}
