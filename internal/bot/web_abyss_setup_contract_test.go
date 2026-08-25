package bot

import (
	"os"
	"strings"
	"testing"
)

func TestAbyssSetupAndNavigationImprovementContracts(t *testing.T) {
	t.Parallel()

	paths := []string{
		"webassets/abyss.html",
		"webassets/abyss_entry_planner.html",
		"webassets/abyss_onboarding.html",
		"webassets/abyss_ui200.css",
	}
	var combined strings.Builder
	for _, path := range paths {
		content, err := webAssets.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		combined.Write(content)
	}
	routes, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	combined.Write(routes)
	source := combined.String()
	for _, required := range []string{
		`/api/abyss/setup/state`, `data-risk=`, `tier_bests`, `floor_one_risk_by_tier`,
		`free_entry_available`, `last_setup`, `Yesterday ·`, `entryJackpot`, `entryFocus`,
		`×0.75 rewards`, `skips floors 1–`, `Cursed bank: +20%`, `animateNumber(el,lastPactSum`,
		`pactConflicts`, `.ab-tier:has(input[value='insanity'])`, `.ab-tier-unlockbar`,
		`ab-tab-dot`, `@media (max-width: 900px)`, `ab-anchor`, `ab-backtop`, `ab_sidebar_order`,
	} {
		if !strings.Contains(source, required) {
			t.Errorf("Abyss setup/navigation contract is missing %q", required)
		}
	}
	for _, forbidden := range []string{"ab_free_entry", "ab_tier_best", "ab_last_session", "abyssEntrySetupV1"} {
		if strings.Contains(source, forbidden) {
			t.Errorf("browser-only authoritative state key remains: %q", forbidden)
		}
	}
}
