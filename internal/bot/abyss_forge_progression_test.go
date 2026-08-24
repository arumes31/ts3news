package bot

import (
	"slices"
	"testing"
)

func TestAbyssForgeMasteryRewardsUnlockProgressively(t *testing.T) {
	cosmetics, unlocks := abyssForgeMasteryRewards(10)
	if !slices.Contains(cosmetics, "bronze artisan frame") || !slices.Contains(cosmetics, "silver anvil flourish") {
		t.Fatalf("level 10 cosmetics = %v", cosmetics)
	}
	if !slices.Contains(unlocks, "expanded craft cap") || !slices.Contains(unlocks, "extra queue preset slots") {
		t.Fatalf("level 10 unlocks = %v", unlocks)
	}
	if got := abyssForgeMilestoneStage(25); got != "Exalted" {
		t.Fatalf("stage at 25 = %q, want Exalted", got)
	}
}
