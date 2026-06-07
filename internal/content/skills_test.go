package content

import (
	"testing"
)

func TestSkillLogic(t *testing.T) {
	s := Skill{Power: 2.0, IgnoreDef: 0.5, StunChance: 0.1, HealPercent: 0.2, Special: EffectPhoenix}
	if s.Score() == 0 {
		t.Error("Skill.Score() should not be zero")
	}
	
	if _, ok := GetSkillByID("S0_1"); !ok {
		t.Error("GetSkillByID(S0_1) failed")
	}
	if _, ok := GetSkillByID("INVALID"); ok {
		t.Error("GetSkillByID(INVALID) should fail")
	}
	
	sRand := RandomSkill()
	if !IsSkill(sRand.Name) {
		t.Errorf("IsSkill(%q) failed", sRand.Name)
	}
	if IsSkill("INVALID") {
		t.Error("IsSkill(INVALID) should fail")
	}
}

func TestUltimateSkillLogic(t *testing.T) {
	us := RandomUltimateSkill()
	if !IsUltimateSkill(us.Name) {
		t.Errorf("IsUltimateSkill(%q) failed", us.Name)
	}
	if IsUltimateSkill("INVALID") {
		t.Error("IsUltimateSkill(INVALID) should fail")
	}
	
	if _, ok := GetUltimateSkillByID(us.ID); !ok {
		t.Errorf("GetUltimateSkillByID(%q) failed", us.ID)
	}
	if _, ok := GetUltimateSkillByID("INVALID"); ok {
		t.Error("GetUltimateSkillByID(INVALID) should fail")
	}
}
