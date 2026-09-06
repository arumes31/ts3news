package bot

import (
	"math"
	"testing"
)

func TestAbyssGoldAddSaturatesWithoutLosingExistingGold(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ current, gain, want int64 }{
		{math.MaxInt64 - 3, 2, math.MaxInt64 - 1},
		{math.MaxInt64 - 3, 4, math.MaxInt64},
		{math.MaxInt64, 0, math.MaxInt64},
		{100, -10, 100},
	} {
		if got := abyssGoldAdd(test.current, test.gain); got != test.want {
			t.Errorf("add(%d, %d) = %d, want %d", test.current, test.gain, got, test.want)
		}
	}
}

func TestAbyssGoldScaleSaturatesAndPreservesIntegerMultipliers(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		amount     int64
		multiplier float64
		want       int64
	}{
		{9_123_939_416_000_000_000, 2, math.MaxInt64},
		{math.MaxInt64 - 1, 1, math.MaxInt64 - 1},
		{1_000_000_000_000_000_001, 2, 2_000_000_000_000_000_002},
		{1_001, 1.5, 1_501},
		{100, math.Inf(1), math.MaxInt64},
		{100, math.NaN(), 0},
	} {
		if got := abyssGoldScale(test.amount, test.multiplier); got != test.want {
			t.Errorf("scale(%d, %v) = %d, want %d", test.amount, test.multiplier, got, test.want)
		}
	}
}

func TestAbyssGoldPercentPreservesHighCacheShares(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		amount  int64
		percent int
		want    int64
	}{
		{9_123_939_416_000_000_000, 25, 2_280_984_854_000_000_000},
		{9_123_939_416_000_000_000, 50, 4_561_969_708_000_000_000},
		{math.MaxInt64, 80, 7_378_697_629_483_820_645},
		{math.MaxInt64, 120, math.MaxInt64},
		{1_001, 15, 150},
	} {
		if got := abyssGoldPercent(test.amount, test.percent); got != test.want {
			t.Errorf("percent(%d, %d) = %d, want %d", test.amount, test.percent, got, test.want)
		}
	}
}
