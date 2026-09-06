package bot

import (
	"math"
	"testing"
)

func TestShopBuffPriceLadder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		owned, want int64
	}{
		{"first", 0, 1_000_000}, {"second", 1, 2_000_000},
		{"before cap", 998, 999_000_000}, {"cap", 999, 1_000_000_000},
		{"after cap", 1000, 1_000_000_000}, {"large counter", math.MaxInt64, 1_000_000_000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := shopBuffPrice(tc.owned); got != tc.want {
				t.Fatalf("price(%d) = %d, want %d", tc.owned, got, tc.want)
			}
		})
	}
}

func TestShopBuffStateUsesSeparatePermanentCounters(t *testing.T) {
	t.Parallel()
	state, err := parseShopBuffState(`{"rarity":42,"quantity":8}`)
	if err != nil {
		t.Fatal(err)
	}
	if state.Rarity != 42 || state.Quantity != 8 || shopBuffPrice(state.Rarity) != 43_000_000 || shopBuffPrice(state.Quantity) != 9_000_000 {
		t.Fatalf("incorrect independent state: %+v", state)
	}
}

func TestShopBuffStateRejectsCorruptCounters(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, raw string }{
		{"invalid JSON", "bad"}, {"null", "null"}, {"negative rarity", `{"rarity":-1,"quantity":0}`},
		{"negative quantity", `{"rarity":0,"quantity":-1}`}, {"fraction", `{"rarity":0.1,"quantity":0}`},
		{"overflow", `{"rarity":9223372036854775808,"quantity":0}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseShopBuffState(tc.raw); err == nil {
				t.Fatalf("accepted corrupt state %s", tc.raw)
			}
		})
	}
}
