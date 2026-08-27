package bot

import (
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssRunLootEstimatedValue(t *testing.T) {
	gear := content.Gear{Rarity: content.RarityRare, Stats: content.Stats{STR: 10}}
	tests := []struct {
		name  string
		grant abyssLootGrant
		want  int64
	}{
		{name: "gold", grant: abyssLootGrant{Type: "gold", Gold: 1234}, want: 1234},
		{name: "tokens", grant: abyssLootGrant{Type: "tokens", Tokens: 2}, want: 2 * int64(abyssTokenBuyGold)},
		{name: "material conversion ladder", grant: abyssLootGrant{Type: "mat", MatID: "prism", MatN: 2}, want: 1_000_000},
		{name: "negative reward", grant: abyssLootGrant{Type: "gold", Gold: -1}, want: 0},
		{name: "bound unlock", grant: abyssLootGrant{Type: "title", TitleName: "Delver"}, want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := abyssRunLootEstimatedValue(test.grant); got != test.want {
				t.Fatalf("abyssRunLootEstimatedValue() = %d, want %d", got, test.want)
			}
		})
	}
	if got := abyssRunLootEstimatedValue(abyssLootGrant{Type: "gear", Gear: &gear}); got != max(gearPrice(gear)/2, int64(1)) {
		t.Fatalf("gear estimate = %d, want vendor quote", got)
	}
}

func TestAbyssCoreInterfaceAssets(t *testing.T) {
	for _, asset := range []string{"webassets/abyss.html", "webassets/abyss_core_interface.html", "webassets/abyss_core_interface.css", "webassets/abyss_combat_recorder.html", "webassets/abyss_forge_workstation.html", "webassets/abyss_inventory_ui.html", "webassets/abyss_longterm.html", "webassets/abyss_loot_signals.html", "webassets/abyss_player_experience.html", "webassets/abyss_stage_hud.html"} {
		body, err := webAssets.ReadFile(asset)
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		for _, marker := range map[string][]string{
			"webassets/abyss.html":                   {`id="threatPct"`, `id="bossDistance"`, `id="lootTypeFilters"`, `data-estimated-value=`, `achievementBannerQueue`, `ab-boss-spawn-shake`, `ab-tier-icon`, `ab-tier-rate`, `ab-cons-state`},
			"webassets/abyss_core_interface.html":    {`function openRarityGuide()`, `function renderInsuranceNudge()`, `window.showAbyssShortcuts`, `function milestoneToast`, `function updateParallax`},
			"webassets/abyss_core_interface.css":     {`.ab-log-timestamps`, `.ab-insurance-attention`, `.ab-loot-type-filters`, `.ab-drop-streak-flame`, `.ab-boss-hp.has-phases`, `.ab-depth-backdrop`, `.ab-milestone-toast`, `.ab-boss-spawn-shake`, `.abyss-stage.ab-downed .ab-escrow-val`, `.ab-manifest-skeleton`},
			"webassets/abyss_combat_recorder.html":   {`id="abyssShortcutHelp"`, `id="logAutoScroll"`},
			"webassets/abyss_forge_workstation.html": {`id="forgeItemSearch"`, `forgeSlotFilter`, `event.key==='ArrowDown'`},
			"webassets/abyss_inventory_ui.html":      {`function setRunLootManifestLoading`, `aria-busy`, `loadingTimer`},
			"webassets/abyss_longterm.html":          {`key:'ab_logtime'`, `key:'ab_loottype'`},
			"webassets/abyss_loot_signals.html":      {`id="dropStreakFlame"`, `data-streak=`},
			"webassets/abyss_player_experience.html": {`key==='ab_fontsize'`, `/api/abyss/preferences/font-size`, `Account font size`},
			"webassets/abyss_stage_hud.html":         {`--boss-hp`, `boss.dataset.phase`, `Phase markers at 50% and 25%`},
		}[asset] {
			if !strings.Contains(text, marker) {
				t.Fatalf("%s missing %q", asset, marker)
			}
		}
	}
}

func TestAbyssTierListIncludesIconsAndAccountRates(t *testing.T) {
	tiers := abyssTierListWithRates(999, []abyssTierRateView{{Tier: "hell", Runs: 8, Wins: 3, Percent: 38}})
	if len(tiers) != len(abyssTierOrder) {
		t.Fatalf("tiers = %d, want %d", len(tiers), len(abyssTierOrder))
	}
	for _, tier := range tiers {
		if tier.Icon == "" {
			t.Errorf("tier %q has no icon", tier.Key)
		}
		if tier.Key == "hell" && tier.WinRateHint != "38% bank rate · 8 runs" {
			t.Errorf("Hell rate = %q", tier.WinRateHint)
		}
	}
}
