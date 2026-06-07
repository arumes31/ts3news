package content

import (
	"testing"
)

func TestHazardLogic(t *testing.T) {
	zType := getZoneTypeFromName("Volcanic Region")
	if zType != ZoneVolcanic {
		t.Errorf("getZoneTypeFromName(Volcanic) = %q", zType)
	}
	
	z := Zone{Name: "Volcanic Region"}
	hazards := GetZoneHazards(z, 1.0)
	if len(hazards) == 0 {
		t.Error("GetZoneHazards should return at least one hazard for Volcanic")
	}
	
	users := []*UserInCombat{{Nickname: "Hero", Stats: Stats{HP: 100}, CurrentHP: 100}}
	mobs := []*Mob{{Name: "Orc", MaxHP: 100, CurrentHP: 100}}
	hEffects := []HazardEffect{
		{Hazard: AllHazards[0], Remaining: 5},
	}
	
	logs := []string{}
	rem := ApplyHazardEffects(users, mobs, hEffects, z, &logs)
	if len(rem) != 1 {
		t.Error("ApplyHazardEffects should return 1 remaining effect")
	}
	if rem[0].Remaining != 4 {
		t.Errorf("Remaining duration = %d, want 4", rem[0].Remaining)
	}
	if users[0].CurrentHP >= 100 {
		t.Error("User should have taken damage from hazard")
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
