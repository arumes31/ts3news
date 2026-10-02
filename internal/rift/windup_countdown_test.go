package rift

import "testing"

func TestEnemyWindupsCountDownWithoutRestarting(t *testing.T) {
	for _, kind := range []string{"goblin", "knight", "archer", "boss"} {
		t.Run(kind, func(t *testing.T) {
			r := testRun()
			r.Player.X, r.Player.Y = 500, 410
			x := 560.0
			if kind == "archer" {
				x = 750
			}
			r.Enemies = []Actor{{Kind: kind, X: x, Y: 410, HP: 100, MaxHP: 100}}
			r.enemyTick(0, .02)
			if r.Enemies[0].Windup == 0 {
				t.Fatal("enemy did not start telegraph")
			}
			for i := 0; i < 70 && r.Enemies[0].Attacks == 0; i++ {
				before := r.Enemies[0].Windup
				r.Player.Y = 410 + float64(i%2)*2
				r.enemyTick(0, .02)
				if r.Enemies[0].Windup >= before {
					t.Fatal("windup restarted or stopped counting down")
				}
			}
			if r.Enemies[0].Attacks != 1 || r.Enemies[0].Windup != 0 || r.Enemies[0].Cooldown <= 0 {
				t.Fatal("telegraph failed to release exactly one attack into recovery")
			}
		})
	}
}
