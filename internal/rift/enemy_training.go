package rift

// EnemyTrainingProfile describes the same base rules used by enemy combat.
type EnemyTrainingProfile struct {
	BossPhases       []BossPhaseTraining `json:"boss_phases,omitempty"`
	WindupSeconds    float64             `json:"windup_seconds"`
	ResistsKnockdown bool                `json:"resists_knockdown"`
	Interruptible    bool                `json:"interruptible"`
}

// EnemyTraining returns the base windup and control resistances for an enemy role.
func EnemyTraining(kind string) EnemyTrainingProfile {
	boss := kind == "boss"
	windup := .55
	if kind == "treasure" {
		windup = 0
	}
	if boss {
		windup = bossPhaseTraining[0].WindupSeconds
	}
	profile := EnemyTrainingProfile{WindupSeconds: windup, ResistsKnockdown: boss, Interruptible: !boss}
	if boss {
		profile.BossPhases = append([]BossPhaseTraining(nil), bossPhaseTraining[:]...)
	}
	return profile
}

// BossPhaseTraining is shared by combat timing and the bestiary phase guide.
type BossPhaseTraining struct {
	Phase           int     `json:"phase"`
	AtHealthPercent int     `json:"at_health_percent"`
	WindupSeconds   float64 `json:"windup_seconds"`
}

var bossPhaseTraining = [...]BossPhaseTraining{{1, 100, 1.15}, {2, 50, .98}, {3, 25, .85}}

func enemyWindup(e *Actor) float64 {
	if e.Kind == "boss" {
		return bossPhaseTraining[max(1, min(3, e.Phase))-1].WindupSeconds
	}
	return EnemyTraining(e.Kind).WindupSeconds
}
