package rift

import (
	"math"
	"testing"
	"time"
)

func TestClassPracticeTargetMechanics(t *testing.T) {
	names := map[string]bool{}
	for _, class := range []string{"vanguard", "berserker", "marksman", "beastmaster", "elementalist", "chronomancer", "oracle", "geomancer", "bloodblade", "voidwalker", "runesmith", "alchemist"} {
		t.Run(class, func(t *testing.T) {
			build := testRun().Build
			build.Class = class
			build.Signatures = []Skill{{ID: "builder", Role: "builder"}, {ID: "finisher", Role: "finisher"}}
			r, err := NewPracticeRun("target", build, "class", time.Unix(100, 0))
			if err != nil {
				t.Fatal(err)
			}
			e := r.Enemies[0]
			if names[e.Name] || r.Practice.TargetHint == "" {
				t.Fatal("missing distinctive target or guidance")
			}
			names[e.Name] = true
			injured := class == "oracle" || class == "bloodblade" || class == "alchemist"
			if injured && r.Player.HP != r.Player.MaxHP*.6 {
				t.Fatal("healing practice has no missing health")
			}
			r.hurtEnemyPiercing(0, 100, "hit", 0)
			ordinary := r.Stats.DamageDealt
			if math.Abs(ordinary-100*(1-e.Armor)) > .001 {
				t.Fatalf("target armor ignored: %v armor %v", ordinary, e.Armor)
			}
			if r.Enemies[0].HP != e.HP {
				t.Fatal("target health condition changed after hit")
			}
			r.hurtEnemyPiercing(0, 100, "hit", 1)
			if math.Abs(r.Stats.DamageDealt-ordinary-100) > .001 {
				t.Fatal("piercing did not bypass armor")
			}
			r.skillHit(0, 100, build.Signatures[1], 1, e.ID)
			if class == "berserker" && math.Abs(r.Stats.DamageDealt-ordinary-100-125) > .001 {
				t.Fatal("wounded target failed to exercise execution bonus")
			}
			if err := r.ResetPractice(time.Unix(110, 0)); err != nil {
				t.Fatal(err)
			}
			if r.Enemies[0].Name != e.Name || r.Enemies[0].HP != e.HP || r.Enemies[0].Armor != e.Armor || r.Practice.ClassHits != 0 {
				t.Fatal("reset did not restore target setup")
			}
			if r.Build.Pets != build.Pets || r.Build.Relic != build.Relic {
				t.Fatal("practice changed equipment")
			}
			if r.Gold != 0 || len(r.Drops) != 0 {
				t.Fatal("target awarded loot")
			}
		})
	}
}
