package bot

import (
	"strings"
	"testing"
)

func TestAbyssCustomStakeBoundaries(t *testing.T) {
	t.Parallel()

	for _, tokens := range []int{0, 5, 10, 20} {
		if !abyssTokenAnteValid(tokens) {
			t.Errorf("valid token ante %d rejected", tokens)
		}
	}
	for _, tokens := range []int{-5, 1, 15, 21, 100} {
		if abyssTokenAnteValid(tokens) {
			t.Errorf("invalid token ante %d accepted", tokens)
		}
	}
	for percent := -20; percent <= 50; percent += 10 {
		if !abyssRiskDialValid(percent) {
			t.Errorf("valid risk dial %d rejected", percent)
		}
	}
	for _, percent := range []int{-30, -15, 5, 55} {
		if abyssRiskDialValid(percent) {
			t.Errorf("invalid risk dial %d accepted", percent)
		}
	}
}

func TestAbyssCustomStakeRewardAndRisk(t *testing.T) {
	t.Parallel()

	if got := applyAbyssStakeReward(1_000, 20, 50); got != 1_800 {
		t.Fatalf("maximum custom-stake reward = %d, want 1800", got)
	}
	if got := applyAbyssStakeReward(1_000, 0, -20); got != 800 {
		t.Fatalf("low-risk reward = %d, want 800", got)
	}
	if got := applyAbyssStakeReward(1_000, 15, 5); got != 1_000 {
		t.Fatalf("invalid custom stakes changed reward to %d", got)
	}
	if got := abyssRiskWithDial(40, 50); got != 60 {
		t.Fatalf("risk forecast = %d, want 60", got)
	}
	if got := abyssRiskWithDial(80, 50); got != 100 {
		t.Fatalf("risk forecast cap = %d, want 100", got)
	}
	if got := abyssRiskDialMultiplier(-20); got != 0.8 {
		t.Fatalf("low-risk danger multiplier = %.2f, want 0.80", got)
	}
}

func TestAbyssCustomStakeSetupCanonicalization(t *testing.T) {
	t.Parallel()

	valid := canonicalAbyssEntrySetup(abyssEntrySetup{Tier: "normal", TokenAnte: 10, RiskDialPct: 30})
	if valid.TokenAnte != 10 || valid.RiskDialPct != 30 {
		t.Fatalf("valid custom stakes changed: %#v", valid)
	}
	invalid := canonicalAbyssEntrySetup(abyssEntrySetup{Tier: "normal", TokenAnte: 11, RiskDialPct: 25})
	if invalid.TokenAnte != 0 || invalid.RiskDialPct != 0 {
		t.Fatalf("invalid custom stakes survived: %#v", invalid)
	}
}

func TestAbyssCustomStakeUIContracts(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	planner, err := webAssets.ReadFile("webassets/abyss_entry_planner.html")
	if err != nil {
		t.Fatal(err)
	}
	risk, err := webAssets.ReadFile("webassets/abyss_core_risk.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{`id="tokenAnte"`, `id="entryRiskDial"`, `token_ante:`, `risk_dial_pct:`, `ante_return`} {
		if !strings.Contains(string(page), required) {
			t.Errorf("Abyss page is missing %q", required)
		}
	}
	for _, required := range []string{`token_ante`, `risk_dial_pct`, `updateAbyssEntryRiskDial`} {
		if !strings.Contains(string(planner), required) {
			t.Errorf("entry planner is missing %q", required)
		}
	}
	for _, required := range []string{`id="runRiskDial"`, `/api/abyss/risk_dial`, `state.token_ante`, `state.risk_dial_pct`} {
		if !strings.Contains(string(risk), required) {
			t.Errorf("risk console is missing %q", required)
		}
	}
}
