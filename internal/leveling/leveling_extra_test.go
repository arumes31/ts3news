package leveling

import (
	"reflect"
	"testing"
)

func TestSubRank(t *testing.T) {
	tests := []struct {
		lvl  int
		want int
	}{
		{1, 1},
		{30, 30},
		{31, 1},
		{0, 1},
		{-5, 1},
	}
	for _, tt := range tests {
		if got := SubRank(tt.lvl); got != tt.want {
			t.Errorf("SubRank(%d) = %d, want %d", tt.lvl, got, tt.want)
		}
	}
}

func TestTierForLevel(t *testing.T) {
	tests := []struct {
		lvl  int
		want int
	}{
		{1, 1},
		{30, 1},
		{31, 2},
		{0, 1},
		{MaxLevel * 2, NumTiers},
	}
	for _, tt := range tests {
		if got := TierForLevel(tt.lvl); got != tt.want {
			t.Errorf("TierForLevel(%d) = %d, want %d", tt.lvl, got, tt.want)
		}
	}
}

func TestTierName(t *testing.T) {
	if TierName(1) != "Drifter" {
		t.Errorf("TierName(1) = %q, want %q", TierName(1), "Drifter")
	}
	// Test procedural name
	name := TierName(NumTiers)
	if name == "" {
		t.Error("TierName(NumTiers) should not be empty")
	}
}

func TestXPPerPoke(t *testing.T) {
	for i := 0; i < 100; i++ {
		xp := XPPerPoke()
		if xp < xpMin || xp > xpMax {
			t.Errorf("XPPerPoke() = %d, out of range [%d, %d]", xp, xpMin, xpMax)
		}
	}
}

func TestXPForPrice(t *testing.T) {
	tests := []struct {
		price   float64
		cheaper bool
		wantMin bool
	}{
		{-10, false, true},
		{100, false, false},
		{0, true, false},
		{60, true, true},
	}
	for _, tt := range tests {
		got := XPForPrice(tt.price, tt.cheaper)
		if tt.wantMin {
			if got != xpMin {
				t.Errorf("XPForPrice(%f, %v) = %d, want %d", tt.price, tt.cheaper, got, xpMin)
			}
		} else {
			if got != xpMax {
				t.Errorf("XPForPrice(%f, %v) = %d, want %d", tt.price, tt.cheaper, got, xpMax)
			}
		}
	}
}

func TestParseLevelGroups(t *testing.T) {
	tests := []struct {
		input string
		want  map[int]int
	}{
		{"1:10, 50:20", map[int]int{1: 10, 50: 20}},
		{"", map[int]int{}},
		{"invalid", map[int]int{}},
		{"1:abc", map[int]int{}},
		{"1:10, , 20:30", map[int]int{1: 10, 20: 30}},
	}
	for _, tt := range tests {
		got := ParseLevelGroups(tt.input)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("ParseLevelGroups(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestMilestonesCrossed(t *testing.T) {
	groups := map[int]int{10: 100, 20: 200, 5: 50}
	tests := []struct {
		old, new int
		want     []int
	}{
		{1, 15, []int{50, 100}},
		{10, 20, []int{200}},
		{20, 10, []int{}},
		{0, 30, []int{50, 100, 200}},
	}
	for _, tt := range tests {
		got := MilestonesCrossed(tt.old, tt.new, groups)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("MilestonesCrossed(%d, %d) = %v, want %v", tt.old, tt.new, got, tt.want)
		}
	}
}

func TestRoman(t *testing.T) {
	if got := roman(0); got != "I" {
		t.Errorf("roman(0) = %q, want %q", got, "I")
	}
}

func TestDeromanInvalid(t *testing.T) {
	if got := deroman("INVALID"); got != 0 {
		t.Errorf("deroman(INVALID) = %d, want 0", got)
	}
}

func TestLevelByNameEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		want int
		ok   bool
	}{
		{"Drifter", 0, false},
		{"Drifter INVALID", 0, false},
		{"UnknownTier I", 0, false},
		{"Drifter of the Void I", 0, false}, // Only valid for levels > 6000
	}
	for _, tt := range tests {
		got, ok := LevelByName(tt.name)
		if ok != tt.ok || got != tt.want {
			t.Errorf("LevelByName(%q) = %d, %v; want %d, %v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}
