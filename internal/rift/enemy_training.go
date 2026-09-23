package rift

// EnemyTrainingProfile describes the same base rules used by enemy combat.
type EnemyTrainingProfile struct {
	WindupSeconds    float64 `json:"windup_seconds"`
	ResistsKnockdown bool    `json:"resists_knockdown"`
	Interruptible    bool    `json:"interruptible"`
}

// EnemyTraining returns the base windup and control resistances for an enemy role.
func EnemyTraining(kind string) EnemyTrainingProfile {
	boss := kind == "boss"
	windup := .55
	if kind == "treasure" {
		windup = 0
	}
	if boss {
		windup = 1.15
	}
	return EnemyTrainingProfile{WindupSeconds: windup, ResistsKnockdown: boss, Interruptible: !boss}
}
