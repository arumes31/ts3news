package content

import (
	"testing"
)

func TestHazardLogic(t *testing.T) {
	// Test all zone type detections
	zoneTests := []struct {
		name string
		want ZoneType
	}{
		{"Volcanic Region", ZoneVolcanic},
		{"Deep Cave", ZoneUnderground},
		{"Hellish Maw", ZoneHell},
		{"Toxic Swamp", ZoneSwamp},
		{"Ancient Ruins", ZoneRuins},
		{"Sandy Desert", ZoneDesert},
		{"Radioactive Wasteland", ZoneWasteland},
		{"Frosty Tundra", ZoneTundra},
		{"High Peak", ZoneMountain},
		{"Coral Beach", ZoneBeach},
		{"Dark Dungeon", ZoneDungeon},
		{"Arcane Spells", ZoneMagic},
		{"Unknown", ZoneDesert},
	}
	for _, tt := range zoneTests {
		if got := getZoneTypeFromName(tt.name); got != tt.want {
			t.Errorf("getZoneTypeFromName(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
	
	z := Zone{Name: "Volcanic Region"}
	hazards := GetZoneHazards(z, 3.0) // High difficulty for more hazards
	if len(hazards) == 0 {
		t.Error("GetZoneHazards should return at least one hazard for Volcanic")
	}
	
	// Test all hazard effect types
	users := []*UserInCombat{
		{Nickname: "Hero", Stats: Stats{HP: 100, STR: 100, DEF: 100, SPD: 100}, CurrentHP: 100},
	}
	mobs := []*Mob{
		{Name: "Orc", MaxHP: 100, CurrentHP: 100, Stats: Stats{STR: 100, DEF: 100, SPD: 100}},
	}
	
	for _, h := range AllHazards {
		hEffects := []HazardEffect{{Hazard: h, Remaining: 2}}
		logs := []string{}
		ApplyHazardEffects(users, mobs, hEffects, z, &logs)
	}
}

func TestResistanceValue(t *testing.T) {
	s := Stats{STA: 1000}
	res := getResistanceValue(s, "STA")
	if res <= 0 || res > 0.75 {
		t.Errorf("getResistanceValue(1000) = %f", res)
	}
}

func TestHazardProtection(t *testing.T) {
	h := AllHazards[0] // Boiling Lava (DamageOverTime, STA resistance)
	gear := GetHazardProtectionGear(h)
	if len(gear) == 0 {
		t.Error("GetHazardProtectionGear returned empty")
	}
	
	cons := GetHazardProtectionConsumable(h)
	if len(cons) == 0 {
		t.Error("GetHazardProtectionConsumable returned empty")
	}
}
