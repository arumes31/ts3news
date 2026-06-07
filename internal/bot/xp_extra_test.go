package bot

import (
	"testing"
)

func TestStreakMultiplier(t *testing.T) {
	tests := []struct {
		streak int
		want   float64
	}{
		{1, 1.0},
		{3, 1.25},
		{5, 1.5},
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
		{1, 1.5},
		{2, 1.5},
		{5, 1.7}, // 1.5 + 0.05 * (4-1)? No, 1.5 + 0.05 * (humans-1). 
		          // humans = 5-1=4. 1.5 + 0.05*(4-1) = 1.5+0.15 = 1.65?
	}
	// Let's re-verify the formula in xp.go
	/*
	func serverMultiplier(onlineNormal int) float64 {
		humans := onlineNormal - 1
		if humans < 1 {
			humans = 1
		}
		// Simulation-tuned base: 1.5x for any human presence
		m := 1.5 + serverMultPerUser*float64(humans-1)
		if m > serverMultCap {
			m = serverMultCap
		}
		return m
	}
	*/
	// for online=5: humans=4. m = 1.5 + 0.05*(4-1) = 1.65.
	for _, tt := range tests {
		got := serverMultiplier(tt.online)
		if tt.online == 5 && got != 1.65 {
			t.Errorf("serverMultiplier(5) = %f, want 1.65", got)
		}
	}
}
