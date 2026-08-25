package bot

import (
	"os"
	"strings"
	"testing"
)

func TestAbyssEscrowSoftCap(t *testing.T) {
	under := applyAbyssEscrowSoftCap(10_000, 500, 2_000, 10)
	if under.Escrow != 12_500 || under.Bonus != 2_000 || under.EfficiencyPct != 100 {
		t.Fatalf("under-cap growth = %#v", under)
	}
	above := applyAbyssEscrowSoftCap(200_000, 4_000, 8_000, 10)
	if above.SoftCap != 150_000 || above.Escrow != 203_000 || above.Bonus != 2_000 || above.EfficiencyPct != 25 {
		t.Fatalf("over-cap growth = %#v", above)
	}
	crossing := applyAbyssEscrowSoftCap(149_000, 2_000, 4_000, 10)
	if crossing.Escrow <= 149_000 || crossing.Escrow >= 155_000 || crossing.EfficiencyPct <= 25 || crossing.EfficiencyPct >= 100 {
		t.Fatalf("cross-cap growth = %#v", crossing)
	}
}

func TestAbyssEstablishedCoreRules(t *testing.T) {
	if abyssLastStandCost(1) != 5 || abyssLastStandCost(25) != 10 {
		t.Fatal("Last Stand cost must scale with depth")
	}
	if got := abyssDepthLootFindBonus(75); got != 0.03 {
		t.Fatalf("depth-75 loot bonus = %.2f", got)
	}
	if got := abyssDepthLootFindBonus(1_000); got != 0.04 {
		t.Fatalf("depth loot bonus cap = %.2f", got)
	}
	if name, aura := abyssPrestigeTier(5); name != "Void Sovereign" || aura == "" {
		t.Fatalf("prestige tier = %q %q", aura, name)
	}
	tier, _ := abyssTierByKey("normal")
	if shallow, deep := abyssRiskPct(1, tier, 500), abyssRiskPct(40, tier, 500); deep <= shallow {
		t.Fatalf("risk did not rise: floor 1=%d floor 40=%d", shallow, deep)
	}
	if abyssComebackEligible(2) || !abyssComebackEligible(3) {
		t.Fatal("comeback must activate at three daily deaths")
	}

	goSource, err := os.ReadFile("web_abyss_features.go")
	if err != nil {
		t.Fatal(err)
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	source := string(goSource) + string(page)
	for _, required := range []string{
		"last_stand_used=TRUE", "bank_locked_floors=2", "revivePct := 25 + 5*st.UpMercy",
		"escrowSoftCap", "escrowEfficiencyPct", "curRisk + '% risk",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("core-rule contract missing %q", required)
		}
	}
}
