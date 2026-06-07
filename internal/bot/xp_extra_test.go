package bot

import (
	"testing"
)

func TestStreakMultiplier(t *testing.T) {
	tests := []struct {
		streak int
		want   float64
	}{
		{0, 1.0},
		{1, 1.0},
		{2, 1.0},
		{3, 1.25},
		{4, 1.25},
		{5, 1.5},
		{6, 1.5},
		{7, 2.0},
		{10, 2.0},
	}
	for _, tt := range tests {
		if got := streakMultiplier(tt.streak); got != tt.want {
			t.Errorf("streakMultiplier(%d) = %f, want %f", tt.streak, got, tt.want)
		}
	}
}

func TestServerMultiplier_Logic(t *testing.T) {
	tests := []struct {
		online int
		want   float64
	}{
		{0, 1.5},
		{1, 1.5},
		{2, 1.5},
		{3, 1.55},
		{5, 1.65},
		{100, serverMultCap},
	}
	for _, tt := range tests {
		got := serverMultiplier(tt.online)
		if got != tt.want {
			t.Errorf("serverMultiplier(%d) = %f, want %f", tt.online, got, tt.want)
		}
	}
}

func TestLootBoxForCross(t *testing.T) {
	tests := []struct {
		old, new int
		wantBox  bool
	}{
		{1, 1, false},
		{24, 25, true},
		{25, 26, false},
		{1, 50, true},
		{49, 51, true},
	}
	for _, tt := range tests {
		box := lootBoxForCross(tt.old, tt.new)
		if (box > 0) != tt.wantBox {
			t.Errorf("lootBoxForCross(%d, %d) = %d, wantBox = %v", tt.old, tt.new, box, tt.wantBox)
		}
	}
}

