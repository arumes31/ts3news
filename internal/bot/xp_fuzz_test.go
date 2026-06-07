package bot

import (
	"testing"
	"ts3news/internal/content"
)

func FuzzGetElementMult(f *testing.F) {
	elements := []string{
		string(content.ElementPhysical),
		string(content.ElementFire),
		string(content.ElementWater),
		string(content.ElementEarth),
		string(content.ElementAir),
	}
	for _, a := range elements {
		for _, d := range elements {
			f.Add(a, d)
		}
	}

	f.Fuzz(func(t *testing.T, attacker, defender string) {
		a := content.Element(attacker)
		d := content.Element(defender)
		mult := getElementMult(a, d)
		if mult <= 0 {
			t.Errorf("getElementMult(%s, %s) returned non-positive multiplier: %f", attacker, defender, mult)
		}
		if mult > 2.0 {
			t.Errorf("getElementMult(%s, %s) returned unexpectedly high multiplier: %f", attacker, defender, mult)
		}
	})
}

func FuzzServerMultiplier(f *testing.F) {
	f.Add(1)
	f.Add(10)
	f.Add(100)
	f.Add(1000)

	f.Fuzz(func(t *testing.T, online int) {
		m := serverMultiplier(online)
		if m < 1.0 {
			t.Errorf("serverMultiplier(%d) returned < 1.0: %f", online, m)
		}
		if m > serverMultCap {
			t.Errorf("serverMultiplier(%d) exceeded cap: %f", online, m)
		}
	})
}

func FuzzFormatGold(f *testing.F) {
	f.Add(int64(0))
	f.Add(int64(100))
	f.Add(int64(1000))
	f.Add(int64(1000000))
	f.Add(int64(1000000000))

	f.Fuzz(func(t *testing.T, gold int64) {
		res := FormatGold(gold)
		if res == "" {
			t.Errorf("FormatGold(%d) returned empty string", gold)
		}
	})
}

func FuzzLootBoxForCross(f *testing.F) {
	f.Add(1, 2)
	f.Add(24, 25)
	f.Add(25, 26)
	f.Add(1, 100)

	f.Fuzz(func(t *testing.T, oldLevel, newLevel int) {
		if oldLevel < 0 || newLevel < 0 {
			return
		}
		box := lootBoxForCross(oldLevel, newLevel)
		if newLevel <= oldLevel && box != 0 {
			t.Errorf("lootBoxForCross(%d, %d) returned %d but level didn't increase", oldLevel, newLevel, box)
		}
		if box < 0 {
			t.Errorf("lootBoxForCross(%d, %d) returned negative XP: %d", oldLevel, newLevel, box)
		}
	})
}
