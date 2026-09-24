package rift

import (
	"math"
	"testing"
)

func TestBeastmasterPetsOnlyScaleChargedFinishersUpToThree(t *testing.T) {
	for pets := 0; pets <= 5; pets++ {
		for _, class := range []string{"beastmaster", "vanguard"} {
			for _, role := range []string{"builder", "finisher"} {
				for _, charges := range []int{0, 1, 3} {
					r := testRun()
					r.Build.Class = class
					r.Build.Pets = pets
					r.Enemies = []Actor{{ID: "target", HP: 1000, MaxHP: 1000}}
					r.skillHit(0, 100, Skill{Role: role, Kind: "fire"}, charges, "")
					want := 100.
					if class == "beastmaster" && role == "finisher" && charges > 0 {
						want *= 1 + float64(min(3, pets))*.1
					}
					if math.Abs((1000-r.Enemies[0].HP)-want) > .00001 {
						t.Fatalf("pets=%d class=%s role=%s charges=%d: damage %v want %v", pets, class, role, charges, 1000-r.Enemies[0].HP, want)
					}
				}
			}
		}
	}
}
