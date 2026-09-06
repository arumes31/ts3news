package bot

import (
	"math"
	"os"
	"strings"
	"testing"
)

func TestAbyssFlawlessContractFailsOnDamage(t *testing.T) {
	t.Parallel()

	flags := map[string]int64{}
	seedAbyssContractPact(flags, "flawless", 0)
	if mult := applyAbyssContractFloor(flags, true); mult != abyssContractRewardMult {
		t.Fatalf("intact multiplier = %v", mult)
	}
	if mult := applyAbyssContractFloor(flags, false); mult != 1 || flags[abyssRunFlagContractFailed] != 1 {
		t.Fatalf("failed multiplier = %v, flags = %#v", mult, flags)
	}
	if fee := abyssContractForfeit(1000, flags, 12, false); fee != 250 {
		t.Fatalf("forfeit = %d, want 250", fee)
	}
	second := map[string]int64{}
	seedAbyssContractPact(second, "flawless", 0)
	failAbyssContractOnDefeat(second)
	if second[abyssRunFlagContractFailed] != 1 {
		t.Fatalf("defeat did not fail flawless contract: %#v", second)
	}
}

func TestAbyssCheckpointContractSettlesAtTarget(t *testing.T) {
	t.Parallel()

	flags := map[string]int64{}
	seedAbyssContractPact(flags, "checkpoint", 12)
	view := abyssContractViewFromFlags(flags, 12)
	if view == nil || view.TargetDepth != 20 || view.Failed {
		t.Fatalf("contract view = %#v", view)
	}
	if fee := abyssContractForfeit(1000, flags, 19, false); fee != 250 {
		t.Fatalf("early forfeit = %d, want 250", fee)
	}
	if fee := abyssContractForfeit(1000, flags, 20, false); fee != 0 {
		t.Fatalf("completed forfeit = %d, want 0", fee)
	}
	if fee := abyssContractForfeit(1000, flags, 19, true); fee != 0 {
		t.Fatalf("partial-bank forfeit = %d, want 0", fee)
	}
}

func TestAbyssContractPactServerAndClientContract(t *testing.T) {
	t.Parallel()

	server, err := os.ReadFile("web_abyss.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"seedAbyssContractPact", "applyAbyssContractFloor", "abyssContractNonCombatRewardMult", "contract_forfeit"} {
		if !strings.Contains(string(server), required) {
			t.Errorf("contract server flow is missing %q", required)
		}
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"contractPact", "Flawless Clause", "Checkpoint Clause", "Failed contract forfeiture"} {
		if !strings.Contains(string(page), required) {
			t.Errorf("contract UI is missing %q", required)
		}
	}
}

func TestAbyssContractForfeitPreservesHighCache(t *testing.T) {
	t.Parallel()
	flags := map[string]int64{abyssRunFlagContract: 1, abyssRunFlagContractFailed: 1}
	if got := abyssContractForfeit(math.MaxInt64, flags, 439, false); got != 2_305_843_009_213_693_951 {
		t.Fatalf("contract forfeit = %d, want 2305843009213693951", got)
	}
}
