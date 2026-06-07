package leveling

import (
	"testing"
)

func FuzzXPLevelRoundTrip(f *testing.F) {
	f.Add(1)
	f.Add(100)
	f.Add(5000)
	f.Add(10000)
	f.Add(1000000)

	f.Fuzz(func(t *testing.T, level int) {
		if level < 1 || level > absoluteMaxLevel {
			return
		}

		xp := XPForLevel(level)
		if xp < 0 {
			t.Errorf("Level %d resulted in negative XP %d", level, xp)
		}

		calcLevel := LevelForXP(xp)
		if calcLevel != level {
			t.Errorf("Round-trip failed: Level %d -> XP %d -> Level %d", level, xp, calcLevel)
		}
	})
}

func FuzzLevelNameRoundTrip(f *testing.F) {
	f.Add(1)
	f.Add(100)
	f.Add(5000)
	f.Add(10000)

	f.Fuzz(func(t *testing.T, level int) {
		if level < 1 || level > MaxLevel {
			return
		}

		name := LevelName(level)
		calcLevel, ok := LevelByName(name)
		if !ok {
			t.Errorf("LevelByName failed for level %d (name: %s)", level, name)
			return
		}

		if calcLevel != level {
			t.Errorf("Round-trip failed: Level %d -> Name %s -> Level %d", level, name, calcLevel)
		}
	})
}

func FuzzRomanRoundTrip(f *testing.F) {
	for i := 1; i <= 3999; i++ {
		f.Add(i)
	}

	f.Fuzz(func(t *testing.T, i int) {
		if i < 1 || i > 3999 {
			return
		}
		r := roman(i)
		back := deroman(r)
		if back != i {
			t.Errorf("Roman round-trip failed for %d: %s -> %d", i, r, back)
		}
	})
}
