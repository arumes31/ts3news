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
		if boss.VolleyFan {
			plan = BossAttackPlan{Name: "Rotating Fan", Kind: "fan", Windup: 1.25, Recovery: 2}
		}
	}
	if boss.LaneSlams && plan.Kind == "slam" {
		plan = BossAttackPlan{Name: "Lane Slam", Kind: "lane_slam", Windup: 1.6, Recovery: 2.3}
	}
	if boss.RingAttack && plan.Kind == "slam" && boss.Attacks%4 == 2 {
		plan = BossAttackPlan{Name: "Void Ring", Kind: "ring", Windup: 1.6, Recovery: 2.3}
	}
	if boss.LaneSlams && boss.Attacks%4 == 3 {
		plan = BossAttackPlan{Name: "Time Pulse", Kind: "channel", Windup: 2, Recovery: 2.3}
	}
	if bossSurgeTurn(boss) {
		plan = BossAttackPlan{Name: "Ground Surge", Kind: "surge", Windup: 2, Recovery: 2.3}
	}
	if r.wantsBossCharge(boss) {
		plan = BossAttackPlan{Name: "Charge", Kind: "charge", Windup: 1.15, Recovery: 2.3}
	}
	if r.Practice != nil && r.Practice.SlowTelegraphs {
		plan.Windup *= 2
	}
	return plan
}
