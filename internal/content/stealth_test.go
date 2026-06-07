package content

import (
	"testing"
)

func TestStealthLogic(t *testing.T) {
	user := &UserInCombat{
		Nickname: "Ninja",
		Equipped: map[GearSlot]Gear{
			SlotChest: {Name: "Shadow Cloak", Special: EffectStealth},
		},
		Skills: []Skill{{Name: "Stealthy Move"}},
	}
	
	zones := []string{"Shadow Forest", "Rocky Mountain", "Open Plain", "Urban City", "Cave", "Swamp", "Desert"}
	for _, zn := range zones {
		zone := Zone{Name: zn}
		state := CalculateStealth(user, zone, "night")
		if state.CurrentStealth <= 0 {
			t.Errorf("Ninja should have high stealth in %s", zn)
		}
		
		mob := &Mob{Level: 10, Stats: Stats{INT: 100}}
		detection := CalculateMobDetection(mob, zone, "day")
		CheckStealthDetection(state, detection)
		ApplyStealthAttack(user, mob, state, false)
	}
	
	if len(GetStealthGear()) == 0 {
		t.Error("GetStealthGear returned empty")
	}
	if len(GetStealthConsumables()) == 0 {
		t.Error("GetStealthConsumables returned empty")
	}
}
