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

func TestFinisherCastingAccentEvent(t *testing.T) {
	findEvent := func(events []Event, kind string) *Event {
		for i := range events {
			if events[i].Kind == kind {
				return &events[i]
			}
		}
		return nil
	}

	r := testRun()
	r.Build.Signatures = []Skill{
		{ID: "build", Role: "builder", Kind: "shield", Cost: 1, Cooldown: 0},
		{ID: "finish", Role: "finisher", Kind: "slash", Cost: 1, Cooldown: 0, Power: 2},
	}
	r.Player.X = 420
	r.Player.Y = 310

	// 1. Builder cast does not emit finisher_cast
	r.cast("build")
	if ev := findEvent(r.Events, "finisher_cast"); ev != nil {
		t.Fatal("builder cast must not emit finisher_cast event")
	}

	// 2. Charged finisher cast emits finisher_cast with exact charge count (1)
	r.Player.Cooldown = 0
	r.Events = nil
	r.cast("finish")
	ev := findEvent(r.Events, "finisher_cast")
	if ev == nil {
		t.Fatal("finisher cast must emit finisher_cast event")
	}
	if ev.Value != 1 {
		t.Fatalf("expected finisher_cast value 1 for 1 charge, got %f", ev.Value)
	}
	if ev.X != r.Player.X || ev.Y != r.Player.Y-35 {
		t.Fatalf("expected coordinates (%f, %f), got (%f, %f)", r.Player.X, r.Player.Y-35, ev.X, ev.Y)
	}

	// 3. 3-Charge finisher cast emits finisher_cast with value 3
	for i := 0; i < 3; i++ {
		r.Player.Cooldown = 0
		r.cast("build")
	}
	if r.Resource != 3 {
		t.Fatalf("expected 3 charges, got %d", r.Resource)
	}
	r.Player.Cooldown = 0
	r.Events = nil
	r.cast("finish")
	ev3 := findEvent(r.Events, "finisher_cast")
	if ev3 == nil {
		t.Fatal("3-charge finisher cast must emit finisher_cast event")
	}
	if ev3.Value != 3 {
		t.Fatalf("expected finisher_cast value 3 for 3 charges, got %f", ev3.Value)
	}

	// 4. Empty finisher cast emits finisher_cast with value 0
	r.Player.Cooldown = 0
	r.Events = nil
	r.cast("finish")
	ev0 := findEvent(r.Events, "finisher_cast")
	if ev0 == nil {
		t.Fatal("empty finisher cast must emit finisher_cast event")
	}
	if ev0.Value != 0 {
		t.Fatalf("expected finisher_cast value 0 for empty finisher, got %f", ev0.Value)
	}
}


