package leveling

import "testing"

func TestLevelForXPMonotonic(t *testing.T) {
	if l := LevelForXP(0); l != 1 {
		t.Errorf("LevelForXP(0) = %d, want 1", l)
	}
	prev := 1
	for xp := 0; xp < XPForLevel(MaxLevel); xp += 1000 {
		l := LevelForXP(xp)
		if l < prev {
			t.Fatalf("level decreased at xp=%d: %d < %d", xp, l, prev)
		}
		if l < 1 || l > MaxLevel {
			t.Fatalf("level %d out of range at xp=%d", l, xp)
		}
		prev = l
	}
	if l := LevelForXP(XPForLevel(MaxLevel) + 1_000_000); l != MaxLevel {
		t.Errorf("huge xp should cap at %d, got %d", MaxLevel, l)
	}
}

func TestXPForLevelRoundTrip(t *testing.T) {
	for _, lvl := range []int{1, 2, 10, 100, 500, 1000} {
		xp := XPForLevel(lvl)
		if got := LevelForXP(xp); got != lvl {
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
	if LevelName(1) != "Peasant I" {
		t.Errorf("LevelName(1) = %q, want %q", LevelName(1), "Peasant I")
	}
	if LevelName(1000) == "" {
		t.Error("LevelName(1000) is empty")
	}
}

func TestTenYearDesign(t *testing.T) {
	const avgXP = float64(xpMin+xpMax) / 2.0
	pokes := float64(XPForLevel(MaxLevel)) / avgXP
	years := pokes / 365.0 // assume ~1 notification/day
	if years < 7 || years > 13 {
		t.Errorf("max level reached in ~%.1f years (%.0f pokes); want ~10", years, pokes)
	}
}

func TestXPForPriceDirection(t *testing.T) {
	// Pricier = more XP (default direction).
	cheap := XPForPrice(5, false)
	pricey := XPForPrice(60, false)
	if !(pricey > cheap) {
		t.Errorf("pricier should give more XP: cheap=%d pricey=%d", cheap, pricey)
	}
	// Inverted direction.
	if XPForPrice(5, true) <= XPForPrice(60, true) {
		t.Error("cheaperMoreXP=true should give cheaper games more XP")
	}
	// Bounds.
	for _, p := range []float64{-5, 0, 30, 60, 999} {
		if x := XPForPrice(p, false); x < xpMin || x > xpMax {
			t.Errorf("XPForPrice(%v) = %d out of [%d,%d]", p, x, xpMin, xpMax)
		}
	}
}

func TestParseLevelGroupsAndMilestones(t *testing.T) {
	groups := ParseLevelGroups("10:7, 25:8 ,bad,100:9")
	if len(groups) != 3 || groups[10] != 7 || groups[25] != 8 || groups[100] != 9 {
		t.Fatalf("ParseLevelGroups wrong: %v", groups)
	}
	crossed := MilestonesCrossed(9, 26, groups)
	if len(crossed) != 2 || crossed[0] != 7 || crossed[1] != 8 {
		t.Errorf("MilestonesCrossed = %v, want [7 8]", crossed)
	}
	if c := MilestonesCrossed(26, 30, groups); len(c) != 0 {
		t.Errorf("MilestonesCrossed should be empty, got %v", c)
	}
}
