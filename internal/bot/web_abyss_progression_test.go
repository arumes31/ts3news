package bot

import (
	"testing"
	"time"
)

func TestAbyssSeasonalTreeIsStableWithinQuarter(t *testing.T) {
	a := abyssSeasonalTree(time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC))
	b := abyssSeasonalTree(time.Date(2026, time.September, 30, 23, 0, 0, 0, time.UTC))
	if a.Key != b.Key || a.Sector != b.Sector {
		t.Fatalf("same quarter changed branch: %#v != %#v", a, b)
	}
	c := abyssSeasonalTree(time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC))
	if c.Key == a.Key || c.Sector == a.Sector {
		t.Fatalf("next quarter did not rotate: %#v -> %#v", a, c)
	}
}

func TestAbyssProgressionXPPercentIsBoundedAndAdditive(t *testing.T) {
	if got := abyssProgressionXPPercent(map[string]int64{abyssRunFlagRestedCharges: 1}); got != 20 {
		t.Fatalf("rested bonus = %d, want 20", got)
	}
	if got := abyssProgressionXPPercent(map[string]int64{abyssRunFlagRestedCharges: 5, abyssRunFlagCatchupCharges: 10}); got != 45 {
		t.Fatalf("combined bonus = %d, want 45", got)
	}
}

func TestAbyssVeteranTrackQualification(t *testing.T) {
	tests := []struct {
		track          string
		depth, hp, max int
		want           bool
	}{
		{"iron", 3, 50, 100, true}, {"iron", 3, 51, 100, false},
		{"untouched", 3, 90, 100, true}, {"untouched", 3, 89, 100, false},
		{"boss", 5, 1, 100, true}, {"boss", 4, 100, 100, false},
	}
	for _, tt := range tests {
		if got := abyssVeteranQualifies(tt.track, tt.depth, tt.hp, tt.max); got != tt.want {
			t.Errorf("%s qualification = %v, want %v", tt.track, got, tt.want)
		}
	}
}

func TestAbyssSanctuaryStage(t *testing.T) {
	if stage, _ := abyssSanctuaryStage(nil); stage != 0 {
		t.Fatalf("empty sanctuary stage = %d", stage)
	}
	if stage, _ := abyssSanctuaryStage(map[string]int{"heal": 2, "repair": 2}); stage != 2 {
		t.Fatalf("four upgrades stage = %d", stage)
	}
	if stage, _ := abyssSanctuaryStage(map[string]int{"heal": 3, "repair": 3, "forge": 1}); stage != 3 {
		t.Fatalf("complete sanctuary stage = %d", stage)
	}
}
