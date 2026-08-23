package bot

import (
	"testing"

	"ts3news/internal/content"
)

// TestInsanityTier verifies the tier above Hell: ×10 rewards, ×20 danger, and
// the unlock gate.
func TestInsanityTier(t *testing.T) {
	tier, ok := abyssTierByKey("insanity")
	if !ok {
		t.Fatal("insanity tier not registered")
	}
	if tier.RewardMult != 10.0 {
		t.Errorf("insanity reward mult = %.1f, want 10", tier.RewardMult)
	}
	if tier.DiffMult != 20.0 {
		t.Errorf("insanity danger mult = %.1f, want 20", tier.DiffMult)
	}
	if abyssTierOrder[len(abyssTierOrder)-1] != "insanity" {
		t.Errorf("insanity must be the last tier, order = %v", abyssTierOrder)
	}
	// Locked below depth 50, unlocked at/above.
	if abyssTierList(49)[3].Unlocked {
		t.Error("insanity should be locked at best depth 49")
	}
	if !abyssTierList(50)[3].Unlocked {
		t.Error("insanity should unlock at best depth 50")
	}
}

// TestInsanityExclusiveGear verifies the insanity pool: reachable by ID, mostly
// carrying a negative trade-off, and never leaking into standard drops/shops.
func TestInsanityExclusiveGear(t *testing.T) {
	ids := []string{
		"INSANITY_EDGE", "INSANITY_CLEAVER", "INSANITY_CROWN", "INSANITY_PLATE",
		"INSANITY_TREADS", "INSANITY_LOOP", "INSANITY_PENDANT", "INSANITY_HEART",
	}
	negative := 0
	for _, id := range ids {
		g, ok := content.GetGearByID(id)
		if !ok {
			t.Fatalf("insanity item %s missing from catalog", id)
		}
		if !content.IsInsanityGearID(g.ID) {
			t.Errorf("%s not recognised as insanity gear", id)
		}
		// Trade-off check: a negative stat line, the Cursed flag, or Fragile.
		hasNeg := g.Cursed || g.Special == content.EffectFragile
		for _, e := range g.BonusEffects {
			if e == content.EffectFragile {
				hasNeg = true
			}
		}
		s := g.Stats
		if s.HP < 0 || s.STR < 0 || s.DEF < 0 || s.SPD < 0 || s.LCK < 0 || s.INT < 0 || s.STA < 0 || s.CRT < 0 || s.DGE < 0 || s.MNA < 0 {
			hasNeg = true
		}
		if hasNeg {
			negative++
		}
	}
	if negative < len(ids)*3/4 {
		t.Errorf("most insanity items should carry a negative trade-off: %d/%d do", negative, len(ids))
	}

	// Standard drops must never surface insanity exclusives.
	for i := 0; i < 500; i++ {
		if g := content.RandomGearDrop(); content.IsInsanityGearID(g.ID) {
			t.Fatalf("RandomGearDrop leaked insanity item %s", g.ID)
		}
	}
	for _, g := range content.GearByMinRarity(content.RarityLegendary) {
		if content.IsInsanityGearID(g.ID) {
			t.Fatalf("GearByMinRarity leaked insanity item %s", g.ID)
		}
	}
	for _, g := range content.ShopStock(42, 20) {
		if content.IsInsanityGearID(g.ID) {
			t.Fatalf("ShopStock leaked insanity item %s", g.ID)
		}
	}

	// The dedicated pool roller returns insanity gear and honours exclusions.
	g := content.RandomInsanityGearDrop()
	if !content.IsInsanityGearID(g.ID) {
		t.Fatalf("RandomInsanityGearDrop returned %s", g.ID)
	}
	owned := map[string]bool{}
	for _, id := range ids {
		owned[id] = true
	}
	// With everything owned the reroll budget is exhausted but the result must
	// still be an insanity item (never a cross-pool fallback surprise).
	if g := content.RandomInsanityGearDropExcluding(owned); !content.IsInsanityGearID(g.ID) {
		t.Fatalf("RandomInsanityGearDropExcluding left the pool: %s", g.ID)
	}
}

// TestForgeRarityGoldMult verifies the rarity-scaled forge gold pricing hits the
// requested 10k–10M+ spread.
func TestForgeRarityGoldMult(t *testing.T) {
	want := map[content.Rarity]int64{
		content.RarityCommon:    100,
		content.RarityUncommon:  100,
		content.RarityRare:      250,
		content.RarityEpic:      500,
		content.RarityLegendary: 1000,
		content.RarityMythic:    2000,
		content.RarityDivine:    5000,
		content.RarityCelestial: 10000,
		content.RarityEternal:   25000,
	}
	for r, w := range want {
		if got := forgeRarityGoldMult(r); got != w {
			t.Errorf("forgeRarityGoldMult(%s) = %d, want %d", r, got, w)
		}
	}
	// Spot-check the spread: a 400-base temper is 10k on Common (floor) and
	// 10M on Eternal.
	if raw := int64(400) * forgeRarityGoldMult(content.RarityCommon); raw < abyssForgeGoldFloor {
		t.Errorf("common temper raw %d below floor %d", raw, int64(abyssForgeGoldFloor))
	}
	if raw := int64(400) * forgeRarityGoldMult(content.RarityEternal); raw != 10_000_000 {
		t.Errorf("eternal temper raw = %d, want 10,000,000", raw)
	}
}
