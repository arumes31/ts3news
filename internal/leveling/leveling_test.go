package leveling

import "testing"

func TestLevelForXPMonotonic(t *testing.T) {
	if l := LevelForXP(0); l != 1 {
		t.Errorf("LevelForXP(0) = %d, want 1", l)
	}
	prev := 1
	for xp := 0; xp < XPForLevel(MaxLevel); xp += 10000 {
		l := LevelForXP(xp)
		if l < prev {
			t.Fatalf("level decreased at xp=%d: %d < %d", xp, l, prev)
		}
		if l < 1 {
			t.Fatalf("level %d out of range at xp=%d", l, xp)
		}
		prev = l
	}
}

func TestInfiniteTiers(t *testing.T) {
	// Names beyond MaxLevel are procedural and non-empty.
	for _, lvl := range []int{10001, 10101, 20000} {
		name := LevelName(lvl)
		if name == "" {
			t.Fatalf("empty procedural name at level %d", lvl)
		}
	}
}

func TestXPForLevelRoundTrip(t *testing.T) {
	for _, lvl := range []int{1, 2, 10, 100, 1000, 10000} {
		xp := XPForLevel(lvl)
		got := LevelForXP(xp)
		if got != lvl {
			t.Errorf("LevelForXP(XPForLevel(%d)=%d) = %d, want %d", lvl, xp, got, lvl)
		}
	}
}

func TestLevelNameCoversAllLevels(t *testing.T) {
	for lvl := 1; lvl <= MaxLevel; lvl++ {
		if name := LevelName(lvl); name == "" {
			t.Fatalf("empty level name at level %d", lvl)
		}
	}
	if LevelName(1) != "Drifter I" {
		t.Errorf("LevelName(1) = %q, want %q", LevelName(1), "Drifter I")
	}
}

func TestTenYearDesign(t *testing.T) {
	const avgXP = 17.5 // (10+25)/2
	pokes := float64(XPForLevel(MaxLevel)) / avgXP
	years := pokes / 365.0 // assume ~1 notification/day
	if years < 7 || years > 15 {
		t.Errorf("max level reached in ~%.1f years (%.0f pokes); want ~10", years, pokes)
	}
}

func TestXPForPriceDirection(t *testing.T) {
	// Pricier = more XP (default direction).
	cheap := XPForPrice(5, false)
	pricey := XPForPrice(60, false)
	if pricey <= cheap {
		t.Errorf("pricier should give more XP: cheap=%d pricey=%d", cheap, pricey)
	}
	// Inverted direction.
	if XPForPrice(5, true) <= XPForPrice(60, true) {
		t.Error("cheaperMoreXP=true should give cheaper games more XP")
	}
}

func TestLevelByName(t *testing.T) {
	for _, lvl := range []int{1, 91, 601, 1501} {
		name := LevelName(lvl)
		got, ok := LevelByName(name)
		if !ok || got != lvl {
			t.Errorf("LevelByName(%q) = %d,%v want %d", name, got, ok, lvl)
		}
	}
}

func TestDeroman(t *testing.T) {
	tests := []struct {
		r string
		n int
	}{
		{"I", 1},
		{"IV", 4},
		{"IX", 9},
		{"XLII", 42},
		{"XC", 90},
		{"CMXCIX", 999},
		{"M", 1000},
	}
	for _, tt := range tests {
		if got := deroman(tt.r); got != tt.n {
			t.Errorf("deroman(%q) = %d, want %d", tt.r, got, tt.n)
		}
	}
}
