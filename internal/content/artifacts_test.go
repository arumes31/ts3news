package content

import (
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
