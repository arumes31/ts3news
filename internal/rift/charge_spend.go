package rift

// ChargeSpend preserves the most recent confirmed resource-consuming cast.
type ChargeSpend struct {
	SkillID   string `json:"skill_id"`
	SkillName string `json:"skill_name"`
	Charges   int    `json:"charges"`
}
