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
	zone := Zone{Name: "Shadow Forest"}
	
	state := CalculateStealth(user, zone, "night")
	if state.CurrentStealth <= 0 {
		t.Error("Ninja should have high stealth")
	}
	
	mob := &Mob{Level: 10, Stats: Stats{INT: 100}}
	detection := CalculateMobDetection(mob, zone, "day")
	if detection.BaseDetection <= 0 {
		t.Error("Mob should have some base detection")
	}
	
	CheckStealthDetection(state, detection)
	
	bonus := ApplyStealthAttack(user, mob, state, false)
	if bonus <= 0 {
		t.Error("Undetected stealth attack should have bonus damage")
	}
	
	if len(GetStealthGear()) == 0 {
		t.Error("GetStealthGear returned empty")
	}
	if len(GetStealthConsumables()) == 0 {
		t.Error("GetStealthConsumables returned empty")
	}
}
