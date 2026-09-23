package rift

import "testing"

func TestEnemyTrainingMatchesActualWindupAndResistance(t *testing.T) {
	for _, kind := range []string{"goblin", "knight", "archer", "treasure", "boss"} {
		r := testRun()
		r.Enemies = []Actor{{ID: "trainer", Kind: kind, X: r.Player.X + 60, Y: r.Player.Y, HP: 100, MaxHP: 100}}
		if kind == "archer" {
			r.Enemies[0].X = r.Player.X + 250
		}
		if kind == "treasure" {
			r.Enemies[0].X = 55
			r.Player.X = 60
		}
		r.enemyTick(0, .01)
		if kind == "treasure" && (r.Enemies[0].HP != 0 || r.Enemies[0].Pose != "escape") {
			t.Fatal("treasure must escape at the boundary instead of winding up")
		}
		profile := EnemyTraining(kind)
		if r.Enemies[0].Windup != profile.WindupSeconds {
			t.Fatalf("%s windup drift: %v versus %v", kind, r.Enemies[0].Windup, profile.WindupSeconds)
		}
		if profile.ResistsKnockdown != (kind == "boss") || profile.Interruptible == (kind == "boss") {
			t.Fatal("training resistance drift")
		}
	}
}
