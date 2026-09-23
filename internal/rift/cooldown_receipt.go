package rift

import "math"

type CooldownRecovery struct {
	Name    string  `json:"name"`
	Seconds float64 `json:"seconds"`
}

type CooldownReceipt struct {
	Source    string             `json:"source"`
	Recovered []CooldownRecovery `json:"recovered"`
}

func (r *Run) rewindCooldowns(skill Skill) {
	receipt := &CooldownReceipt{Source: skill.Name, Recovered: []CooldownRecovery{}}
	recover := func(id, name string) {
		before := r.SkillTimers[id]
		r.SkillTimers[id] = math.Max(0, before-1.5)
		if before > r.SkillTimers[id] {
			receipt.Recovered = append(receipt.Recovered, CooldownRecovery{Name: name, Seconds: math.Min(1.5, before)})
		}
	}
	for _, ability := range r.abilities() {
		if ability.ID != skill.ID {
			recover(ability.ID, ability.Name)
		}
	}
	recover("jump", "Jump")
	r.LastCooldownReceipt = receipt
}
