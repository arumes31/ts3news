package rift

import (
	"math"
	"testing"
)

func TestBerserkerFuryHealthBoundariesAndArmor(t *testing.T) {
	for _, tc := range []struct {
		class  string
		hp     float64
		active bool
	}{{"berserker", 30, true}, {"berserker", 29, true}, {"berserker", 30.001, false}, {"berserker", 0, false}, {"vanguard", 20, false}} {
		r := testRun()
		r.Build.Class = tc.class
		r.Player.MaxHP = 100
		r.Player.HP = tc.hp
		r.Enemies = []Actor{{ID: "target", HP: 1000, MaxHP: 1000, Armor: .2, ArtKey: "target"}}
		if r.BerserkerFury() != tc.active {
			t.Fatalf("wrong state for %+v", tc)
		}
		r.hurtEnemy(0, 100, "hit")
		want := 80.
		if tc.active {
			want = 92
		}
		if math.Abs(1000-r.Enemies[0].HP-want) > .00001 {
			t.Fatalf("damage for %+v was %v", tc, 1000-r.Enemies[0].HP)
		}
	}
}

func TestBerserkerFuryCombinesWithFinisherAndStopsAfterHealing(t *testing.T) {
	r := testRun()
	r.Build.Class = "berserker"
	r.Player.MaxHP = 100
	r.Player.HP = 30
	r.Enemies = []Actor{{ID: "target", HP: 500, MaxHP: 1000, ArtKey: "target"}}
	r.skillHit(0, 100, Skill{Role: "finisher", Kind: "slash"}, 1, "")
	if math.Abs(500-r.Enemies[0].HP-143.75) > .00001 {
		t.Fatal("finisher and fury multiplier mismatch")
	}
	r.Player.HP = 31
	hp := r.Enemies[0].HP
	r.hurtEnemy(0, 100, "hit")
	if hp-r.Enemies[0].HP != 100 {
		t.Fatal("fury persisted after healing")
	}
}
