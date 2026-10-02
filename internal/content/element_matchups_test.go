package content

import "testing"

func TestElementMatchupsRemainCanonical(t *testing.T) {
	elements := []Element{ElementPhysical, ElementFire, ElementWater, ElementEarth, ElementAir, "", "unknown", "fire"}
	strong := map[Element]Element{ElementFire: ElementAir, ElementAir: ElementEarth, ElementEarth: ElementWater, ElementWater: ElementFire}
	weak := map[Element]Element{ElementFire: ElementWater, ElementWater: ElementEarth, ElementEarth: ElementAir, ElementAir: ElementFire}
	for _, attack := range elements {
		if got := ElementWeakness(attack); got != weak[attack] {
			t.Fatalf("weakness %q = %q", attack, got)
		}
		for _, defense := range elements {
			want := 1.0
			if target, ok := strong[attack]; ok && target == defense {
				want = 2
			}
			if target, ok := weak[attack]; ok && target == defense {
				want = .5
			}
			if got := ElementMultiplier(attack, defense); got != want {
				t.Fatalf("%q against %q: got %v want %v", attack, defense, got, want)
			}
		}
	}
}
