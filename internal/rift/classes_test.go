package rift

import "testing"

func TestClassBuilderAndFinisherResource(t *testing.T) {
	for _, style := range []string{"vanguard", "berserker", "marksman", "beastmaster", "elementalist", "chronomancer", "oracle", "geomancer", "bloodblade", "voidwalker", "runesmith", "alchemist"} {
		t.Run(style, func(t *testing.T) {
			r := testRun()
			r.Build.Class = style
			r.Build.Signatures = []Skill{{ID: "build", Role: "builder", Kind: "shield", Cost: 1, Cooldown: 0}, {ID: "finish", Role: "finisher", Kind: "slash", Cost: 1, Cooldown: 0, Power: 2}}
			for i := 0; i < 4; i++ {
				r.Player.Cooldown = 0
				r.cast("build")
			}
			if r.Resource != 3 {
				t.Fatalf("resource cap: %d", r.Resource)
			}
			r.Player.Cooldown = 0
			r.cast("finish")
			if r.Resource != 0 {
				t.Fatal("finisher did not spend resource")
			}
		})
	}
}

func TestClassPayoffsAndNonlethalHealthCost(t *testing.T) {
	r := testRun()
	r.Build.Class = "voidwalker"
	r.Player.HP = 1
	r.Resource = 3
	r.Build.Signatures = []Skill{{ID: "finish", Role: "finisher", Kind: "void", Cost: 1, Power: 2}}
	r.cast("finish")
	if r.Player.HP < 1 {
		t.Fatal("void finisher killed caster")
	}
	r = testRun()
	r.Build.Class = "chronomancer"
	r.Resource = 2
	r.SkillTimers["fire"] = 5
	r.Build.Signatures = []Skill{{ID: "finish", Role: "finisher", Kind: "void", Cost: 1, Power: 2}}
	r.cast("finish")
	if r.SkillTimers["fire"] != 3.5 {
		t.Fatal("temporal release did not recover cooldown")
	}
	r = testRun()
	r.Build.Class = "bloodblade"
	r.Player.HP = 100
	r.Resource = 2
	r.Build.Signatures = []Skill{{ID: "finish", Role: "finisher", Kind: "slash", Cost: 1, Power: 2}}
	r.cast("finish")
	if r.Player.HP <= 100 {
		t.Fatal("bloodblade finisher did not heal")
	}
}

func TestResourceGainEvents(t *testing.T) {
	r := testRun()
	r.Build.Signatures = []Skill{{ID: "build", Role: "builder", Kind: "shield", Cost: 1, Cooldown: 0}}
	r.cast("build")

	foundResource := false
	for _, e := range r.Events {
		if e.Kind == "resource" && e.Value == 1 {
			foundResource = true
			break
		}
	}
	if !foundResource {
		t.Fatal("expected resource event on builder cast")
	}

	r.Player.HP = 50
	r.healPlayer(20)
	foundHeal := false
	for _, e := range r.Events {
		if e.Kind == "heal" && e.Value == 20 {
			foundHeal = true
			break
		}
	}
	if !foundHeal {
		t.Fatal("expected heal event on healPlayer")
	}

	r.addBarrier(25, "barrier_skill")
	foundBarrier := false
	for _, e := range r.Events {
		if e.Kind == "barrier" && e.Value == 25 {
			foundBarrier = true
			break
		}
	}
	if !foundBarrier {
		t.Fatal("expected barrier event on addBarrier")
	}
}

