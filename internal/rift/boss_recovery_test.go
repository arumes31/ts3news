package rift

import (
	"encoding/json"
	"testing"
)

func TestBossAreaRecoverySurvivesSaveAndInterrupt(t *testing.T) {
	for _, shared := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "catalog"}[shared], func(t *testing.T) {
			r := testRun()
			r.Player.X = 500
			r.Player.Y = 410
			boss := Actor{ID: "boss", Kind: "boss", X: 550, Y: 410, HP: 1000, MaxHP: 1000, Windup: .01, TargetX: 500, TargetY: 410, Phase: 1}
			if shared {
				boss.ArtKey = "monster:test"
			}
			r.Enemies = []Actor{boss}
			r.enemyTick(0, .02)
			if r.Enemies[0].Cooldown != 2.3 || r.Enemies[0].Attacks != 1 {
				t.Fatal("slam did not begin full recovery")
			}
			r.hurtEnemy(0, 600, "hit")
			if r.Enemies[0].Phase < 2 || r.Enemies[0].Cooldown != 2.3 {
				t.Fatal("phase transition shortened recovery")
			}
			r.skillHit(0, 1, Skill{Kind: "quake"}, 0, "")
			if r.Enemies[0].Cooldown != 2.3 {
				t.Fatal("stagger shortened recovery")
			}
			data, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var restored Run
			if err = json.Unmarshal(data, &restored); err != nil {
				t.Fatal(err)
			}
			for range 22 {
				restored.enemyTick(0, .1)
			}
			if restored.Enemies[0].Windup != 0 || restored.Enemies[0].Attacks != 1 {
				t.Fatal("boss chained an attack before recovery ended")
			}
			restored.enemyTick(0, .11)
			if restored.Enemies[0].Windup <= 0 {
				t.Fatal("boss failed to resume after recovery")
			}
			if restored.Enemies[0].Attacks != 1 {
				t.Fatal("boss skipped its next windup")
			}
		})
	}
}
