package bot

import (
	"os"
	"strings"
	"testing"
)

func TestAbyssInsuranceLoyalty(t *testing.T) {
	base := abyssInsuranceCost(10_000, 50, 0, 0)
	loyal := abyssInsuranceCost(10_000, 50, 0, 10_000_000)
	if loyal >= base {
		t.Fatalf("loyal premium = %d, want below base %d", loyal, base)
	}
	if got := abyssInsuranceLoyaltyPct(99_000_000); got != 15 {
		t.Fatalf("loyalty discount = %d%%, want capped 15%%", got)
	}

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"lifetimeBanked", "loyaltyPct", "loyalty_discount_pct"} {
		if !strings.Contains(string(page), required) {
			t.Errorf("insurance loyalty UI missing %q", required)
		}
	}
}

func TestAbyssGuaranteedRestCadence(t *testing.T) {
	if abyssRestFloorDue(3, 9) {
		t.Fatal("rest became due before seven floors elapsed")
	}
	if !abyssRestFloorDue(3, 10) {
		t.Fatal("rest must be due after seven floors")
	}
	if abyssRestFloorDue(40, 41) {
		t.Fatal("checkpoint entry must start a fresh rest cadence")
	}

	source, err := os.ReadFile("web_abyss.go")
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../db/migrations/0073_abyss_rest_cadence.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"last_rest_depth", "abyssRestFloorDue(run.LastRestDepth, newDepth)", "CASE WHEN $2='rest' THEN $1"} {
		if !strings.Contains(string(source)+string(migration), required) {
			t.Errorf("authoritative rest cadence missing %q", required)
		}
	}
}
