package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAbyssLayoutReadabilityProgramContracts(t *testing.T) {
	t.Parallel()

	root := abyssAAARepositoryRoot(t)
	checks := map[string][]string{
		"internal/bot/webassets/abyss.html": {
			`id="miniHud"`, `data-abyss-section`, `id="bossDistance"`, `id="threatPct"`,
			`data-depth="{{.Depth}}"`, `id="lootTypeFilters"`, `data-estimated-value`,
			`activeRunPacts.forEach`, `confirmModal`, `ab-empty-state`,
		},
		"internal/bot/webassets/abyss_navigation.html": {
			"initCollapsiblePanels", "localStorage.setItem(key,open?'1':'0')", "initShortcuts", "ab-compact",
		},
		"internal/bot/webassets/abyss_mobile.html": {
			"initMobileForgeAccordions", "initMobileTierSelect", "initLandscapeLog",
		},
		"internal/bot/webassets/abyss_combat_recorder.html": {
			`id="abyssShortcutHelp"`,
		},
		"internal/bot/webassets/abyss_accessibility.html": {
			"applyAbyssLogSize", "abyssMobileActions", "confirmModal('Swipe gesture selected",
		},
		"internal/bot/webassets/abyss_core_interface.html": {
			"openRarityGuide", "renderInsuranceNudge", "renderDepthPresentation",
		},
		"internal/bot/webassets/abyss_core_interface.css": {
			"ab-manifest-skeleton", "ab-log-timestamps", "ab-tier-rate", "ab-cons-state",
		},
		"internal/bot/webassets/abyss_item_numbers.html": {
			"Numbers: compact", "Exact value:", "localStorage.setItem(storageKey,mode)",
		},
		"internal/bot/webassets/abyss_longterm.html": {
			"Abyss settings", "Run loot filter", "Combat log timestamps", "Compact veteran layout",
		},
		"internal/bot/webassets/abyss_forge_workstation.html": {
			`id="forgeItemSearch"`, "Search forge items", "forgePickerCount", "Select '+item.name+' for forging",
		},
		"internal/bot/webassets/abyss_layout_readability.html": {
			"Run loot row density", "syncPrepConsumables",
			"two-round potion-belt cooldown", "renderLiveCombat",
		},
	}
	for relative, required := range checks {
		raw, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		content := string(raw)
		for _, token := range required {
			if !strings.Contains(content, token) {
				t.Errorf("%s is missing %q", relative, token)
			}
		}
	}
}
