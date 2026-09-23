package rift

// BossAttackPlan describes the next committed attack, before Attacks increments.
// Cooldown, attacker limits and obstruction may delay or cancel its execution.
type BossAttackPlan struct {
	Name     string
	Kind     string
	Windup   float64
	Recovery float64
}

// NextBossAttack exposes the same attack sequence and timings used by combat.
// It is pure, so replay tools can inspect a saved actor without advancing it.
func (r *Run) NextBossAttack(boss Actor) BossAttackPlan {
	if boss.Kind != "boss" {
		return BossAttackPlan{}
	}
	plan := BossAttackPlan{Name: r.bossAttackName(&boss), Kind: "slam", Windup: enemyWindup(&boss), Recovery: 2.3}
	if boss.ArtKey != "" && (boss.Attacks+1)%2 == 0 {
		plan.Kind, plan.Recovery = "projectile", 1.6
	}
	return plan
}
