package bot

import (
	"math"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssRuneResonanceMatchesAttackElementOnce(t *testing.T) {
	t.Parallel()

	equipped := map[content.GearSlot]content.Gear{
		content.SlotMainHand: {Rune: string(content.ElementFire)},
		content.SlotOffHand:  {Rune: string(content.ElementFire)},
	}
	if !abyssRuneResonates(equipped, content.ElementFire) {
		t.Fatal("matching offensive rune did not resonate")
	}
	if abyssRuneResonates(equipped, content.ElementWater) {
		t.Fatal("mismatched offensive rune resonated")
	}
	if got := applyAbyssRuneResonance(2, equipped, content.ElementFire); math.Abs(got-2.1) > 0.0001 {
		t.Fatalf("resonant multiplier = %.2f, want 2.10", got)
	}
}

func TestAbyssRuneResonanceIgnoresWardsAndPetGear(t *testing.T) {
	t.Parallel()

	equipped := map[content.GearSlot]content.Gear{
		content.SlotChest: {Rune: content.DefensiveRuneName(content.ElementFire)},
		content.SlotPet1:  {Rune: string(content.ElementFire)},
	}
	if abyssRuneResonates(equipped, content.ElementFire) {
		t.Fatal("defensive or pet rune activated offensive resonance")
	}
	if abyssRuneResonates(equipped, "") {
		t.Fatal("empty attack element activated resonance")
	}
}
