package content

// ElementMultiplier is the canonical Abyss elemental damage matchup. Unknown
// elements, physical attacks and equal elements are neutral.
func ElementMultiplier(attacker, defender Element) float64 {
	// Fire > Air > Earth > Water > Fire
	switch attacker {
	case ElementFire:
		if defender == ElementAir {
			return 2.0
		}
		if defender == ElementWater {
			return 0.5
		}
	case ElementAir:
		if defender == ElementEarth {
			return 2.0
		}
		if defender == ElementFire {
			return 0.5
		}
	case ElementEarth:
		if defender == ElementWater {
			return 2.0
		}
		if defender == ElementAir {
			return 0.5
		}
	case ElementWater:
		if defender == ElementFire {
			return 2.0
		}
		if defender == ElementEarth {
			return 0.5
		}
	}
	return 1.0
}

// ElementWeakness returns the element with advantage against the defender.
// Physical and unrecognized elements have no elemental weakness.
func ElementWeakness(element Element) Element {
	switch element {
	case ElementFire:
		return ElementWater
	case ElementWater:
		return ElementEarth
	case ElementEarth:
		return ElementAir
	case ElementAir:
		return ElementFire
	default:
		return ""
	}
}
