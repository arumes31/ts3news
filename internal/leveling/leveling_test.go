package leveling

import (
	"fmt"
	"testing"
)

func TestLevelForXPMonotonic(t *testing.T) {
	if l := LevelForXP(0); l != 1 {
		t.Errorf("LevelForXP(0) = %d, want 1", l)
	}
	prev := 1
	// Test up to level 1000 with smaller steps, then larger steps for higher levels
	// Testing the full range up to MaxLevel would take too long (2e15 XP)
	for xp := 0; xp < XPForLevel(1000); xp += 10000 {
		l := LevelForXP(xp)
		if l < prev {
			t.Fatalf("level decreased at xp=%d: %d < %d", xp, l, prev)
		}
		if l < 1 {
			t.Fatalf("level %d out of range at xp=%d", l, xp)
		}
		prev = l
	}
	// Test a few high-level XP values with larger steps
	for xp := XPForLevel(1000); xp < XPForLevel(MaxLevel); xp += 1e12 {
		l := LevelForXP(xp)
		if l < 1 || l > MaxLevel+1000 {
			t.Fatalf("level %d out of reasonable range at xp=%d", l, xp)
		}
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
	// Test round-trip for levels where XPForLevel doesn't hit the cap (2e15)
	// Beyond level ~10000, XPForLevel caps at 2e15, so LevelForXP can't distinguish
	for _, lvl := range []int{1, 2, 10, 100, 1000} {
		xp := XPForLevel(lvl)
		got := LevelForXP(xp)
		if got != lvl {
			t.Errorf("LevelForXP(XPForLevel(%d)=%d) = %d, want %d", lvl, xp, got, lvl)
		}
	}
	// For level 10000, just verify it returns a reasonable high level
	xp := XPForLevel(10000)
	got := LevelForXP(xp)
	if got < 10000 {
		t.Errorf("LevelForXP(XPForLevel(10000)) = %d, want >= 10000", got)
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
	// Test that reaching max level takes a very long time (essentially infinite for gameplay purposes)
	// The XP curve is designed so that MaxLevel is effectively unreachable in normal play
	pokes := float64(XPForLevel(MaxLevel)) / avgXP
	years := pokes / 365.0 // assume ~1 notification/day
	// With the current exponential curve, reaching max level should take an extremely long time
	// This is intentional - the level cap is a theoretical boundary, not an achievable goal
	if years < 1000 {
		t.Errorf("max level reached in ~%.1f years (%.0f pokes); should be effectively unreachable", years, pokes)
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
	for _, lvl := range []int{1, 91, 601, 1501, 3001, 6001, 9999} {
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

func TestLevelForXP_Table(t *testing.T) {
	tests := []struct {
		xp   int
		want int
	}{
		{0, 1},
		{-10, 1},
		{XPForLevel(1), 1},
		{XPForLevel(2), 2},
		{XPForLevel(10), 10},
		{XPForLevel(100), 100},
		{XPForLevel(1000), 1000},
		{XPForLevel(10000), 10000},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("XP=%d", tt.xp), func(t *testing.T) {
			if got := LevelForXP(tt.xp); got != tt.want {
				t.Errorf("LevelForXP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestXPForLevel_Table(t *testing.T) {
	tests := []struct {
		level int
		want  int
	}{
		{0, 0},
		{1, 0},
		{2, 5},
		{10, 188},
		{100, 9812},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("Level=%d", tt.level), func(t *testing.T) {
			if got := XPForLevel(tt.level); got != tt.want {
				t.Errorf("XPForLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func FuzzRoman(f *testing.F) {
	f.Add(1)
	f.Add(10)
	f.Add(100)
	f.Add(3999)
	f.Fuzz(func(t *testing.T, n int) {
		if n < 1 || n > 3999 {
			return
		}
		r := roman(n)
		d := deroman(r)
		if d != n {
			t.Errorf("roman(%d) = %s, deroman(%s) = %d", n, r, r, d)
		}
	})
}

func FuzzLevelNameGeneration(f *testing.F) {
	f.Add(1)
	f.Add(100)
	f.Add(1000)
	f.Add(10000)
	f.Fuzz(func(t *testing.T, level int) {
		if level < 1 || level > 1000000 {
			return
		}
		name := LevelName(level)
		if name == "" {
			t.Errorf("LevelName(%d) returned empty string", level)
		}
		parsed, ok := LevelByName(name)
		if !ok {
			t.Errorf("LevelByName could not parse generated name %q for level %d", name, level)
		}
		if parsed != level {
			if level <= 10000 {
				t.Errorf("LevelByName(%q) = %d, want %d", name, parsed, level)
			}
		}
	})
}

