package rift

import (
	"encoding/json"
	"testing"
)

func TestBossAttackPlanMatchesReplay(t *testing.T) {
	for _, art := range []string{"", "monster:test"} {
		for phase, windup := range []float64{1.15, .98, .85} {
			r := testRun()
			r.Player.X, r.Player.Y = 500, 410
			r.Enemies = []Actor{{ID: "boss", Kind: "boss", ArtKey: art, Shot: "fire", X: 560, Y: 410, HP: 1000, MaxHP: 1000, Phase: phase + 1}}
			for attack := 0; attack < 4; attack++ {
				plan := r.NextBossAttack(r.Enemies[0])
				kind, name, recovery := "slam", "Mossbound Slam", 2.3
				if art != "" && attack%2 == 1 {
					kind, name, recovery = "projectile", "Cinder Volley", 1.6
				}
				if plan.Kind != kind || plan.Name != name || plan.Windup != windup || plan.Recovery != recovery {
					t.Fatalf("art=%q phase=%d attack=%d plan=%+v", art, phase+1, attack, plan)
				}
				r.enemyTick(0, .02)
				if r.Enemies[0].AttackName != plan.Name || r.Enemies[0].Windup != plan.Windup {
					t.Fatal("windup differs from plan")
				}
				data, err := json.Marshal(r)
				if err != nil {
					t.Fatal(err)
				}
				var restored Run
				if err = json.Unmarshal(data, &restored); err != nil {
					t.Fatal(err)
				}
				r = &restored
				if r.NextBossAttack(r.Enemies[0]) != plan {
					t.Fatal("save changed plan")
				}
				r.enemyTick(0, plan.Windup-.01)
				if r.Enemies[0].Attacks != attack {
					t.Fatal("attack released early")
				}
				before := len(r.Projectiles)
				r.Player.Jump = .5
				r.enemyTick(0, .02)
				if r.Enemies[0].Attacks != attack+1 || r.Enemies[0].Cooldown != plan.Recovery {
					t.Fatal("release differs from plan")
				}
				if (len(r.Projectiles) > before) != (kind == "projectile") {
					t.Fatal("wrong attack kind")
				}
				r.enemyTick(0, plan.Recovery-.01)
				if r.Enemies[0].Windup != 0 {
					t.Fatal("next attack began before recovery ended")
				}
			}
		}
	}
}

func TestNonBossHasNoBossAttackPlan(t *testing.T) {
	r := testRun()
	if r.NextBossAttack(Actor{Kind: "archer"}) != (BossAttackPlan{}) {
		t.Fatal("ordinary enemy received boss plan")
	}
}
