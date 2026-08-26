package content

import (
	"math/rand/v2"
	"strings"
	"testing"
)

func TestRarity(t *testing.T) {
	if RarityCommon.String() != "Common" {
		t.Errorf("RarityCommon.String() = %q", RarityCommon.String())
	}
	if Rarity(-1).String() != "Rarity(-1)" {
		t.Errorf("invalid rarity string = %q", Rarity(-1).String())
	}
	if RarityCommon.Color() == "" {
		t.Error("rarity color empty")
	}
	if Rarity(-1).Color() != "#ffffff" {
		t.Errorf("invalid rarity color = %q", Rarity(-1).Color())
	}
}

func TestStats(t *testing.T) {
	s1 := Stats{HP: 10, STR: 5}
	s2 := Stats{HP: 20, DEF: 3}
	sum := s1.Add(s2)
	if sum.HP != 30 || sum.STR != 5 || sum.DEF != 3 {
		t.Errorf("Stats.Add failed: %+v", sum)
	}
	if s1.Score() == 0 {
		t.Error("Stats.Score() should not be zero")
	}
	scaled := s1.Scaled(2.0)
	if scaled.HP != 20 || scaled.STR != 10 {
		t.Errorf("Stats.Scaled failed: %+v", scaled)
	}
}

func TestGearCombatRating(t *testing.T) {
	g := Gear{Rarity: RarityCommon, Stats: Stats{STR: 10, DEF: 10}}
	cr := g.CombatRating()
	if cr == 0 {
		t.Error("CombatRating should not be zero")
	}
	g2 := Gear{Rarity: RarityLegendary, Stats: Stats{STR: 10, DEF: 10}}
	if g2.CombatRating() <= cr {
		t.Errorf("Legendary gear should have higher CR than Common: %f <= %f", g2.CombatRating(), cr)
	}
}

func TestArtifact(t *testing.T) {
	a := Artifact{Name: "Test", Mult: 1.5}
	if !a.IsBoon() {
		t.Error("Mult 1.5 should be a boon")
	}
	if !strings.Contains(a.XPBonusDesc(), "+50%") {
		t.Errorf("XPBonusDesc = %q", a.XPBonusDesc())
	}
	a2 := Artifact{Mult: 0.5}
	if a2.IsBoon() {
		t.Error("Mult 0.5 should not be a boon")
	}
	if !strings.Contains(a2.XPBonusDesc(), "-50%") {
		t.Errorf("XPBonusDesc = %q", a2.XPBonusDesc())
	}
	if a.Score() == 0 {
		t.Error("Artifact.Score() should not be zero")
	}
}

func TestTitleScore(t *testing.T) {
	ti := Title{XPMultiplier: 2.0, DoubleLoot: true}
	if ti.Score() == 0 {
		t.Error("Title.Score() should not be zero")
	}
}

func TestGetters(t *testing.T) {
	if _, ok := GetGearByID("B_Head"); !ok {
		t.Error("GetGearByID(B_Head) failed")
	}
	if _, ok := GetGearByID("INVALID"); ok {
		t.Error("GetGearByID(INVALID) should fail")
	}
	if _, ok := GetEnchantmentByID("E0"); !ok {
		t.Error("GetEnchantmentByID(E0) failed")
	}
	if _, ok := GetConsumableByID("small_health_potion"); !ok {
		t.Error("GetConsumableByID(small_health_potion) failed")
	}
	// Titles are randomized, so we check if IsTitle works on one we know exists or just generic check
	tName := RandomTitle().Name
	if !IsTitle(tName) {
		t.Errorf("IsTitle(%q) failed", tName)
	}
	if IsTitle("INVALID") {
		t.Error("IsTitle(INVALID) should fail")
	}
	aName := RandomArtifact().Name
	if _, ok := GetArtifactByName(aName); !ok {
		t.Errorf("GetArtifactByName(%q) failed", aName)
	}
	if _, ok := GetArtifactByName("INVALID"); ok {
		t.Error("GetArtifactByName(INVALID) should fail")
	}
	if IsGearOrArtifact("INVALID") {
		t.Error("IsGearOrArtifact(INVALID) should fail")
	}
}

func TestRandomGenerators(t *testing.T) {
	RandomItemEffect()
	RandomConsumable()
	RandomGearDrop()
	g := RandomStarterGear()
	if g.ID == "" {
		t.Error("RandomStarterGear returned gear with empty ID")
	}
	RandomArtifact()
	RandomEnchantment()
	RandomTitle()
}

func TestGoblinKingIsDirectOnlyTitle(t *testing.T) {
	title, ok := GetTitleByName("Goblin King")
	if !ok {
		t.Fatal("GetTitleByName(Goblin King) failed")
	}
	if title.XPMultiplier != 1.10 || title.Stats.LCK != 50 || title.Stats.CHA != 100 {
		t.Fatalf("Goblin King changed: %#v", title)
	}
	for _, candidate := range positiveTitles {
		if candidate.Name == "Goblin King" {
			t.Fatal("Goblin King must not be in the random positive-title pool")
		}
	}
}

func TestRepairKitIIExceedsDefaultKit(t *testing.T) {
	base, ok := GetConsumableByID("repair_kit")
	if !ok {
		t.Fatal("default repair kit missing")
	}
	upgraded, ok := GetConsumableByID("repair_kit_ii")
	if !ok {
		t.Fatal("Repair Kit II missing")
	}
	if upgraded.EffectValue <= base.EffectValue {
		t.Fatalf("Repair Kit II effect %v must exceed default %v", upgraded.EffectValue, base.EffectValue)
	}
}

func TestRandomAbyssGearDropForCategoryExcluding(t *testing.T) {
	for _, category := range []string{"weapon", "armor", "jewelry"} {
		for range 20 {
			gear := RandomAbyssGearDropForCategoryExcluding(category, nil)
			isWeapon := gear.Slot == SlotMainHand || gear.Slot == SlotOffHand || gear.Slot == SlotRanged
			isArmor := gear.Slot == SlotHead || gear.Slot == SlotChest || gear.Slot == SlotLegs || gear.Slot == SlotFeet || gear.Slot == SlotHands || gear.Slot == SlotWaist || gear.Slot == SlotBack
			isJewelry := gear.Slot == SlotNeck || gear.Slot == SlotFinger1 || gear.Slot == SlotFinger2 || gear.Slot == SlotTrinket1 || gear.Slot == SlotTrinket2
			if (category == "weapon" && !isWeapon) || (category == "armor" && !isArmor) || (category == "jewelry" && !isJewelry) {
				t.Fatalf("category %q returned %s in slot %s", category, gear.Name, gear.Slot)
			}
		}
	}
}

func TestGearAppearanceCatalogIsCompleteAndDetached(t *testing.T) {
	t.Parallel()

	catalog := GearAppearanceCatalog()
	if len(catalog) == 0 {
		t.Fatal("appearance catalog is empty")
	}
	seen := make(map[string]bool, len(catalog))
	for _, gear := range catalog {
		if seen[gear.ID] {
			t.Fatalf("duplicate appearance ID %q", gear.ID)
		}
		seen[gear.ID] = true
		if _, ok := GetGearByID(gear.ID); !ok {
			t.Fatalf("appearance %q is not accepted by GetGearByID", gear.ID)
		}
	}

	original, ok := GetGearByID(catalog[0].ID)
	if !ok {
		t.Fatalf("catalog item %q disappeared", catalog[0].ID)
	}
	catalog[0].Name = "mutated"
	if current, _ := GetGearByID(original.ID); current.Name != original.Name {
		t.Fatal("appearance catalog mutation changed global content")
	}
}

func TestAbyssGearCatalogIsExclusiveAndDetached(t *testing.T) {
	t.Parallel()

	catalog := AbyssGearCatalog()
	if len(catalog) == 0 {
		t.Fatal("Abyss gear catalog is empty")
	}
	seen := make(map[string]bool, len(catalog))
	for _, gear := range catalog {
		if !IsAbyssGearID(gear.ID) || IsInsanityGearID(gear.ID) {
			t.Fatalf("non-Abyss gear %q leaked into catalog", gear.ID)
		}
		if seen[gear.ID] {
			t.Fatalf("duplicate Abyss gear ID %q", gear.ID)
		}
		seen[gear.ID] = true
	}

	original := AbyssGearCatalog()[0]
	catalog[0].Name = "mutated"
	catalog[0].BonusEffects = append(catalog[0].BonusEffects, EffectThorns)
	catalog[0].Gemstones = append(catalog[0].Gemstones, "mutated")
	if current := AbyssGearCatalog()[0]; current.Name != original.Name {
		t.Fatal("Abyss gear catalog mutation changed canonical content")
	}
	current := AbyssGearCatalog()[0]
	if len(current.BonusEffects) != len(original.BonusEffects) {
		t.Fatal("Abyss gear bonus-effect mutation changed canonical content")
	}
	if len(current.Gemstones) != len(original.Gemstones) {
		t.Fatal("Abyss gear gemstone mutation changed canonical content")
	}
}

func TestRandomGearDropForSlotsExcludingKeepsPoolAndSlot(t *testing.T) {
	tests := []struct {
		name  string
		pool  GearDropPool
		valid func(Gear) bool
	}{
		{name: "standard", pool: GearDropPoolStandard, valid: func(gear Gear) bool { return !IsAbyssGearID(gear.ID) && !IsInsanityGearID(gear.ID) }},
		{name: "abyss", pool: GearDropPoolAbyss, valid: func(gear Gear) bool { return IsAbyssGearID(gear.ID) }},
		{name: "insanity", pool: GearDropPoolInsanity, valid: func(gear Gear) bool { return IsInsanityGearID(gear.ID) }},
		{name: "starter", pool: GearDropPoolStarter, valid: func(gear Gear) bool {
			for _, candidate := range starterGear {
				if candidate.ID == gear.ID {
					return true
				}
			}
			return false
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			slots := GearDropSlots(test.pool)
			if len(slots) == 0 {
				t.Fatal("pool has no eligible slots")
			}
			wantSlot := slots[0]
			gear, ok := RandomGearDropForSlotsExcludingWithRandom(test.pool, []GearSlot{wantSlot}, nil, rand.New(rand.NewPCG(78, 1)))
			if !ok {
				t.Fatal("targeted roll failed")
			}
			if gear.Slot != wantSlot || !test.valid(gear) {
				t.Fatalf("targeted roll = %#v, want slot %s in %s pool", gear, wantSlot, test.name)
			}
		})
	}
}

func TestRandomGearDropForSlotsExcludingPrefersUnownedCandidate(t *testing.T) {
	slots := GearDropSlots(GearDropPoolAbyss)
	if len(slots) == 0 {
		t.Fatal("Abyss pool has no slots")
	}
	wantSlot := slots[0]
	owned := make(map[string]bool)
	var unownedID string
	for _, gear := range abyssExclusiveGear {
		if gear.Slot != wantSlot {
			continue
		}
		if unownedID == "" {
			unownedID = gear.ID
			continue
		}
		owned[gear.ID] = true
	}
	if unownedID == "" {
		t.Fatal("Abyss pool has no matching candidate")
	}
	gear, ok := RandomGearDropForSlotsExcludingWithRandom(GearDropPoolAbyss, []GearSlot{wantSlot}, owned, rand.New(rand.NewPCG(78, 2)))
	if !ok || owned[gear.ID] {
		t.Fatalf("targeted roll = %#v, want an unowned candidate", gear)
	}
}
