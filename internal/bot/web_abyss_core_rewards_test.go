package bot

import (
	"strings"
	"testing"
	"time"
)

func TestAbyssCoreRiskRewardMath(t *testing.T) {
	t.Parallel()

	if got := abyssGreedyInterestRate(0.05, 29); got != 0.09 {
		t.Fatalf("greedy interest = %v, want 0.09", got)
	}
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	if got := abyssIdleDangerPct(now.Add(-90*time.Minute), now); got != 50 {
		t.Fatalf("idle danger cap = %d, want 50", got)
	}
	if got := abyssFranticBankFee(1_000_000, 149, 1_000); got != 50_000 {
		t.Fatalf("frantic fee = %d, want 50000", got)
	}
	if got := abyssFranticBankFee(1_000_000, 150, 1_000); got != 0 {
		t.Fatalf("fee at threshold = %d, want 0", got)
	}
	for _, valid := range []int{10, 15, 50, 85, 90} {
		if !abyssInsurancePercentValid(valid) {
			t.Errorf("insurance rejected valid percentage %d", valid)
		}
	}
	for _, invalid := range []int{0, 9, 12, 91, 100} {
		if abyssInsurancePercentValid(invalid) {
			t.Errorf("insurance accepted invalid percentage %d", invalid)
		}
	}
	if !abyssCheapskateEligible(5, 101) || abyssCheapskateEligible(5, 100) {
		t.Fatal("cheapskate eligibility does not enforce a strict 5% premium boundary")
	}
	removed, tokens := abyssRestCacheConversion(1_000_000)
	if removed != 500_000 || tokens != 3 {
		t.Fatalf("cache conversion = (%d, %d), want (500000, 3)", removed, tokens)
	}
	if got := abyssDeathWishFloorReward(400, true); got != 800 {
		t.Fatalf("death-wish reward = %d, want 800", got)
	}
	if got := abyssAnchorRefund(100, 1_000, true); got != 500 {
		t.Fatalf("anchor refund = %d, want 500", got)
	}
	if got := abyssEchoBankSeed(1_000_000, false); got != 50_000 {
		t.Fatalf("echo seed = %d, want 50000", got)
	}
	if got := abyssEchoBankSeed(1_000_000, true); got != 100_000 {
		t.Fatalf("double echo seed = %d, want 100000", got)
	}
}

func TestAbyssCoreRiskGuards(t *testing.T) {
	t.Parallel()

	if abyssBankNeedsSafeWord(1_000_000, true) {
		t.Fatal("safe word should apply only above one million")
	}
	if !abyssBankNeedsSafeWord(1_000_001, true) {
		t.Fatal("safe word missing above one million")
	}
	if abyssBankNeedsSafeWord(2_000_000, false) {
		t.Fatal("disabled safe-word preference was ignored")
	}
	if got := normalizeAbyssSafeWord("  bank\t"); got != "BANK" {
		t.Fatalf("normalized safe word = %q, want BANK", got)
	}
	run := abyssRun{Depth: 20}
	base := abyssLastStandCost(run.Depth)
	if cost, available := abyssLastStandOffer(run, nil); cost != base || !available {
		t.Fatalf("first Last Stand = (%d, %v), want (%d, true)", cost, available, base)
	}
	run.LastStandUsed = true
	if cost, available := abyssLastStandOffer(run, map[string]int64{}); cost != base*3 || !available {
		t.Fatalf("second Last Stand = (%d, %v), want (%d, true)", cost, available, base*3)
	}
	if _, available := abyssLastStandOffer(run, map[string]int64{abyssRunFlagSecondLastStandUsed: 1}); available {
		t.Fatal("third Last Stand was offered")
	}
}

func TestAbyssCoreRiskAssetsAndTabs(t *testing.T) {
	t.Parallel()

	module, err := webAssets.ReadFile("webassets/abyss_core_risk.html")
	if err != nil {
		t.Fatalf("read risk module: %v", err)
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	navigation, err := webAssets.ReadFile("webassets/abyss_navigation.html")
	if err != nil {
		t.Fatalf("read navigation module: %v", err)
	}
	joined := string(module) + string(page) + string(navigation)
	for _, contract := range []string{
		`min="{{.CoreLoop.InsuranceMin}}"`,
		`/api/abyss/downed_timeout`,
		`/api/abyss/rest_cache_shrink`,
		`bankSafeWordToggle`,
		`deathWishToggle`,
		`anchorRuneBtn`,
		`data-abyss-section="shop"`,
		`data-abyss-section="forge"`,
		`{key:'shop',label:'🜲 Shop'}`,
		`{key:'forge',label:'⚒️ Forge'}`,
	} {
		if !strings.Contains(joined, contract) {
			t.Errorf("Abyss core UI is missing %q", contract)
		}
	}
}
