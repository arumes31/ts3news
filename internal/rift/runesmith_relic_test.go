package rift

import (
	"math"
	"testing"
)

func TestRunesmithRelicOnlyBoostsChargedFinisherDamage(t *testing.T) {
	for _, relic := range []bool{false, true} {
		for _, charges := range []int{0, 2} {
			r := testRun()
			r.Build.Class = "runesmith"
			r.Build.Relic = relic
			r.Build.Armor = 10
			r.Barrier = 0
			r.Player.MaxHP = 200
			r.Enemies = []Actor{{ID: "target", HP: 1000, MaxHP: 1000}}
			r.skillHit(0, 100, Skill{Role: "finisher", Kind: "rune"}, charges, "")
			want := 100.
			if relic && charges > 0 {
				want = 115
			}
			if math.Abs(1000-r.Enemies[0].HP-want) > .00001 {
				t.Fatal("unexpected relic damage")
			}
			r.Resource = charges
			r.classCast(Skill{ID: "finish", Role: "finisher"})
			wantBarrier := 0.
			if charges > 0 {
				wantBarrier = 35
			}
			if r.Barrier != wantBarrier {
				t.Fatal("relic changed class barrier")
			}
		}
	}
}
