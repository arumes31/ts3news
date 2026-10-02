package rift

import "testing"

func TestElementalReactionConfirmsOnlyMarkedChargedHits(t *testing.T) {
	for _, tc := range []struct {
		class, role, mark string
		charges           int
		want              bool
	}{{"elementalist", "finisher", "target", 1, true}, {"elementalist", "finisher", "other", 1, false}, {"elementalist", "finisher", "target", 0, false}, {"elementalist", "builder", "target", 1, false}, {"marksman", "finisher", "target", 1, false}} {
		r := testRun()
		r.Build.Class = tc.class
		r.Enemies = []Actor{{ID: "target", HP: 1000, MaxHP: 1000, X: 300, Y: 400, Elevation: 16}}
		r.skillHit(0, 100, Skill{Role: tc.role, Kind: "fire"}, tc.charges, tc.mark)
		count := 0
		for _, event := range r.Events {
			if event.Kind == "elemental_reaction" {
				count++
				if event.X != 300 || event.Y != 355 || event.Elevation != 16 || event.Value != 20 {
					t.Fatal("wrong reaction origin")
				}
			}
		}
		if (count == 1) != tc.want {
			t.Fatalf("unexpected reaction for %+v: %d", tc, count)
		}
		if tc.want && r.Enemies[0].HP != 880 {
			t.Fatal("reaction changed existing multiplier")
		}
	}
}
