package content

import (
	"math"
	"testing"
)

func TestTalentEffectiveLevelSoftCap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level int
		want  float64
	}{
		{level: -1, want: 0},
		{level: 0, want: 0},
		{level: 5, want: 5},
		{level: 6, want: 5.5},
		{level: 10, want: 7.5},
		{level: 11, want: 7.5},
	}
	for _, test := range tests {
		if got := TalentEffectiveLevel(test.level); got != test.want {
			t.Errorf("TalentEffectiveLevel(%d) = %v, want %v", test.level, got, test.want)
		}
	}
}

func TestTalentBonusUsesDiminishingRanks(t *testing.T) {
	t.Parallel()

	percentTalent := DeepDelverTalents[0]
	statTalent := DeepDelverTalents[1]
	tests := []struct {
		level   int
		wantPct float64
		wantSTR int
	}{
		{level: 5, wantPct: 0.10, wantSTR: 20},
		{level: 6, wantPct: 0.11, wantSTR: 22},
		{level: 10, wantPct: 0.15, wantSTR: 30},
	}
	for _, test := range tests {
		bonus := TalentBonus(map[string]int{
			percentTalent.Key: test.level,
			statTalent.Key:    test.level,
		}, "")
		if got := bonus.Pct["str_pct"]; math.Abs(got-test.wantPct) > 1e-9 {
			t.Errorf("level %d str_pct = %v, want %v", test.level, got, test.wantPct)
		}
		if bonus.Stats.STR != test.wantSTR {
			t.Errorf("level %d STR = %d, want %d", test.level, bonus.Stats.STR, test.wantSTR)
		}
	}
}
