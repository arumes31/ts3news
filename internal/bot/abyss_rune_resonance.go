package bot

import "ts3news/internal/content"

const abyssRuneResonanceMultiplier = 1.05

// abyssRuneResonates reports whether an offensive rune matches the element of
// the current attack. Resonance is deliberately binary: duplicate matching
// runes do not stack, defensive wards do not count, and pet gear is isolated.
func abyssRuneResonates(equipped map[content.GearSlot]content.Gear, attackElement content.Element) bool {
	if attackElement == "" {
		return false
	}
	for slot, gear := range equipped {
		if content.IsPetGearSlot(slot) {
			continue
		}
		if content.Element(gear.Rune) == attackElement {
			return true
		}
	}
	return false
}

func applyAbyssRuneResonance(
	multiplier float64,
	equipped map[content.GearSlot]content.Gear,
	attackElement content.Element,
) float64 {
	if abyssRuneResonates(equipped, attackElement) {
		return multiplier * abyssRuneResonanceMultiplier
	}
	return multiplier
}
