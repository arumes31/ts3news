package bot

import (
	"math"
	"testing"
)

func TestShopBuffTotalPrice(t *testing.T) {
	for _, tc := range []struct {
		owned, amount, want int64
	}{
		{0, 1, 1_000_000},
		{0, 10, 55_000_000},
		{5, 3, 21_000_000},
		{0, 1001, 501_500_000_000},
		{998, 3, 2_999_000_000},
		{999, 3, 3_000_000_000},
		{10000, 3, 3_000_000_000},
		{math.MaxInt64 - 1, 1, 1_000_000_000},
		{1000, math.MaxInt64 / 1_000_000_000, 9_223_372_036_000_000_000},
	} {
		got, err := shopBuffTotalPrice(tc.owned, tc.amount)
		if err != nil || got != tc.want {
			t.Errorf("price(%d, %d) = %d, %v; want %d", tc.owned, tc.amount, got, err, tc.want)
		}
	}
	for _, tc := range []struct{ owned, amount int64 }{
		{-1, 1}, {0, 0}, {0, -1}, {math.MaxInt64, 1},
		{math.MaxInt64 - 1, 2}, {0, math.MaxInt64},
		{1000, math.MaxInt64/1_000_000_000 + 1},
	} {
		if _, err := shopBuffTotalPrice(tc.owned, tc.amount); err == nil {
			t.Errorf("price(%d, %d) accepted invalid or overflowing purchase", tc.owned, tc.amount)
		}
	}
}
