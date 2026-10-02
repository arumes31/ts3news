package rift

import (
	"testing"
	"ts3news/internal/content"
)

func TestEveryCatalogBossOpeningHasSafeCounterplay(t *testing.T) {
	bosses := 0
	for _, mob := range content.AbyssMobCatalog() {
		boss := AdaptMonster(mob)
		if boss.Kind != "boss" {
			continue
		}
		bosses++
		for _, avoid := range []string{"jump", "leave-area"} {
			t.Run(mob.Name+"/"+avoid, func(t *testing.T) {
				r := testRun()
				r.Player.X, r.Player.Y = 500, 410
				boss.ID = "opening-boss"
				boss.X, boss.Y = 560, 410
				r.Enemies = []Actor{boss}
				hp := r.Player.HP
				warning := r.NextBossAttack(boss).Windup
				r.enemyTick(0, .02)
				if warning < EnemyTraining("boss").WindupSeconds || r.Enemies[0].Windup != warning || r.Enemies[0].AttackName == "" {
					t.Fatal("opening lacks a full named telegraph")
				}
				if r.Enemies[0].TargetX != 500 || r.Enemies[0].TargetY != 410 {
					t.Fatal("opening did not mark the player's starting position")
				}
				r.enemyTick(0, .6)
				if r.Player.HP != hp || r.Enemies[0].Attacks != 0 {
					t.Fatal("opening damaged player before warning ended")
				}
				if avoid == "jump" {
					r.Player.Jump = .5
				} else {
					r.Player.Y = 480
				}
				r.enemyTick(0, warning-.6+.01)
				if r.Player.HP != hp || r.Enemies[0].Attacks != 1 || len(r.Projectiles) != 0 {
					t.Fatal("opening did not allow advertised slam avoidance")
				}
				if r.Enemies[0].Cooldown != 2.3 {
					t.Fatal("opening lacks counterattack recovery")
				}
				r.Player.X, r.Player.Y = 500, 410
				r.Player.Facing = 1
				r.Player.Jump = 0
				before := r.Enemies[0].HP
				r.tick(Input{Attack: true}, .02)
				if r.Enemies[0].HP >= before || r.Player.HP != hp {
					t.Fatal("player could not safely counterattack during recovery")
				}
			})
		}
	}
	if bosses == 0 {
		t.Fatal("catalog contained no bosses")
	}
	t.Logf("verified jump and movement counterplay for %d shared bosses", bosses)
}
