package rift

import "math"

type BloodRecovery struct {
	SkillID   string  `json:"skill_id"`
	SkillName string  `json:"skill_name"`
	Healed    float64 `json:"healed"`
	Overflow  float64 `json:"overflow"`
}

func (r *Run) recordBloodRecovery(skillID string, amount, healed float64) {
	if r.Build.Class != "bloodblade" || skillID == "" || amount <= 0 {
		return
	}
	name := skillID
	for _, skill := range r.abilities() {
		if skill.ID == skillID {
			name = skill.Name
			break
		}
	}
	r.LastBloodRecovery = &BloodRecovery{SkillID: skillID, SkillName: name, Healed: healed, Overflow: math.Max(0, amount-healed)}
}
