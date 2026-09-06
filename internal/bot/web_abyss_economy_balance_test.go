package bot

import (
	"fmt"
	"testing"
)

func TestAbyssEconomyLongRunDoesNotCompoundWithoutBound(t *testing.T) {
	for _, tier := range abyssTierOrder {
		t.Run(tier, func(t *testing.T) {
			var cache int64
			for depth := 1; depth <= 400; depth++ {
				bonus := abyssGoldScale(abyssFloorBonus(depth, 700), abyssTiers[tier].RewardMult)
				cache = abyssNonCombatReward(cache, bonus, abyssGreedyInterestRate(abyssEffectiveInterest(20, true), depth), depth, false).Escrow
				if depth%100 == 0 {
					t.Logf("floor=%d cache=%d", depth, cache)
				}
			}
			if cache > 100_000_000 {
				t.Fatalf("400-floor cache=%d exceeds 100M budget", cache)
			}
		})
	}
}

func TestAbyssEconomyLegacyCacheCannotMintInterestOnExcess(t *testing.T) {
	for _, cache := range []int64{100_000_000, 42_438_666_400_000_000} {
		t.Run(fmt.Sprint(cache), func(t *testing.T) {
			growth := abyssNonCombatReward(cache, 0, 0.23, 400, false)
			if gain := growth.Escrow - cache; gain > 250_000 || gain < 0 {
				t.Fatalf("legacy cache minted %d interest in one floor", gain)
			}
		})
	}
}
