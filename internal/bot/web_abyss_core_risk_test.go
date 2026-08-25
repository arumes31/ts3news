package bot

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestAbyssCoreRiskRewardCalculations(t *testing.T) {
	t.Parallel()

	if got := abyssGreedyInterestRate(0.005, 30); got < 0.0649 || got > 0.0651 {
		t.Fatalf("greedy interest rate = %.4f, want 0.065", got)
	}
	if got := abyssHeavyPocketsPct(1_100_000); got != 20 {
		t.Fatalf("heavy-pockets cap = %d, want 20", got)
	}
	if got := abyssFranticBankFee(2_000_000, 14, 100); got != 100_000 {
		t.Fatalf("frantic fee = %d, want 100000", got)
	}
	if got := abyssFranticBankFee(2_000_000, 15, 100); got != 0 {
		t.Fatalf("15%% HP must not pay a frantic fee, got %d", got)
	}
	if got := abyssDeathWishFloorReward(750, true); got != 1_500 {
		t.Fatalf("death-wish reward = %d, want 1500", got)
	}
	if got := abyssAnchorRefund(100, 1_000, true); got != 500 {
		t.Fatalf("anchor refund = %d, want 500", got)
	}
	if got := abyssAnchorRefund(750, 1_000, true); got != 750 {
		t.Fatalf("anchor reduced a better insurance refund to %d", got)
	}
	if got := abyssEchoBankSeed(1_000_000, false); got != 50_000 {
		t.Fatalf("echo seed = %d, want 50000", got)
	}
	if got := abyssEchoBankSeed(1_000_000, true); got != 100_000 {
		t.Fatalf("double-bank echo seed = %d, want 100000", got)
	}
	removed, tokens := abyssRestCacheConversion(1_000_000)
	if removed != 500_000 || tokens != 3 {
		t.Fatalf("rest conversion = (%d, %d), want (500000, 3)", removed, tokens)
	}
}

func TestAbyssCoreRiskBoundaries(t *testing.T) {
	t.Parallel()

	for percent := 10; percent <= 90; percent += 5 {
		if !abyssInsurancePercentValid(percent) {
			t.Errorf("valid insurance percentage %d rejected", percent)
		}
	}
	for _, percent := range []int{-5, 0, 5, 11, 91, 100} {
		if abyssInsurancePercentValid(percent) {
			t.Errorf("invalid insurance percentage %d accepted", percent)
		}
	}
	if abyssBankNeedsSafeWord(1_000_000, true) {
		t.Fatal("exactly 1M unexpectedly requires the safe word")
	}
	if !abyssBankNeedsSafeWord(1_000_001, true) {
		t.Fatal("payout above 1M did not require the safe word")
	}
	if abyssBankNeedsSafeWord(2_000_000, false) {
		t.Fatal("disabled safe-word preference was ignored")
	}
	if got := normalizeAbyssSafeWord(" bank "); got != "BANK" {
		t.Fatalf("normalized safe word = %q", got)
	}

	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	if got := abyssIdleDangerPct(now.Add(-59*time.Second), now); got != 0 {
		t.Fatalf("sub-minute idle danger = %d", got)
	}
	if got := abyssIdleDangerPct(now.Add(-61*time.Second), now); got != 1 {
		t.Fatalf("one-minute idle danger = %d", got)
	}
	if got := abyssIdleDangerPct(now.Add(-2*time.Hour), now); got != 50 {
		t.Fatalf("idle danger cap = %d", got)
	}
	if got := abyssIdleDangerPct(now.Add(time.Minute), now); got != 0 {
		t.Fatalf("future action timestamp produced danger %d", got)
	}
}

func TestAbyssCoreRiskUIAndAuthorityContracts(t *testing.T) {
	t.Parallel()

	pageBytes, err := os.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	moduleBytes, err := os.ReadFile("webassets/abyss_core_risk.html")
	if err != nil {
		t.Fatal(err)
	}
	serverBytes, err := os.ReadFile("web_abyss.go")
	if err != nil {
		t.Fatal(err)
	}
	econBytes, err := os.ReadFile("web_abyss_econ.go")
	if err != nil {
		t.Fatal(err)
	}
	page, module, server, econ := string(pageBytes), string(moduleBytes), string(serverBytes), string(econBytes)
	for _, required := range []string{
		`template "abyssCoreInsurance"`, `template "abyssCoreRiskConsole"`,
		`/static/abyss_core_risk.css`, `abyssBankCommit(cursed,percent,safeWord,doubleBank)`,
	} {
		if !strings.Contains(page, required) {
			t.Errorf("Abyss page missing %q", required)
		}
	}
	for _, required := range []string{
		`type="range"`, `InsuranceMin`, `InsuranceMax`, `InsuranceStep`,
		`/api/abyss/bank_confirm_toggle`, `/api/abyss/death_wish`,
		`/api/abyss/anchor_rune`, `/api/abyss/rest_cache_shrink`,
		`replaceChildren`, `textContent`,
	} {
		if !strings.Contains(module, required) {
			t.Errorf("core-risk module missing %q", required)
		}
	}
	for _, removed := range []string{`onclick="abyssInsure(25)"`, `onclick="abyssInsure(50)"`, `onclick="abyssInsure(75)"`} {
		if strings.Contains(page, removed) {
			t.Errorf("fixed insurance control remains: %q", removed)
		}
	}
	for _, required := range []string{`"frantic_fee"`, `"requires_safe_word"`, `saveAbyssEchoSeed(tx`, `abyssEchoSeedKey(uid)`} {
		if !strings.Contains(server, required) {
			t.Errorf("bank/entry authority path missing %q", required)
		}
	}
	for _, required := range []string{`abyssAnchorRefund`, `saveRunFlags(tx`, `DELETE FROM abyss_active`} {
		if !strings.Contains(econ, required) {
			t.Errorf("forfeit authority path missing %q", required)
		}
	}
}
