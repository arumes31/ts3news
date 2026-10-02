package rift

import "testing"

func TestOracleHealingDiscardsOverflowButBuildsGrace(t *testing.T) {
	for _, hp := range []float64{50, 95, 100} {
		for _, charges := range []int{0, 3} {
			r := testRun()
			r.Build.Class = "oracle"
			r.Player.MaxHP = 100
			r.Player.HP = hp
			r.Resource = charges
			r.Barrier = 0
			r.Build.Signatures = []Skill{{ID: "mend", Name: "Mending Light", Role: "builder", Kind: "heal", Heal: .15, Cost: 0}}
			r.Player.Cooldown = 0
			r.cast("mend")
			want := min(100., hp+15)
			if r.Player.HP != want || r.Stats.Healing != want-hp || r.Barrier != 0 {
				t.Fatalf("wrong healing overflow at hp=%v: hp=%v healed=%v barrier=%v", hp, r.Player.HP, r.Stats.Healing, r.Barrier)
			}
			if r.Resource != min(3, charges+1) {
				t.Fatal("full health prevented charge gain or exceeded cap")
			}
		}
	}
}
