package bot

import (
	"strings"
	"testing"
)

func TestAbyssStageHUDContracts(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	for _, templateName := range []string{"abyssStageHUD", "abyssStageHUDJS"} {
		if server.tmpl.Lookup(templateName) == nil {
			t.Fatalf("template %q is missing", templateName)
		}
	}

	assets := []string{
		"webassets/abyss.html",
		"webassets/abyss_run_awareness.html",
		"webassets/abyss_stage_hud.html",
		"webassets/abyss_stage_hud.css",
		"webassets/abyss_ui200.css",
	}
	var source strings.Builder
	for _, asset := range assets {
		raw, err := webAssets.ReadFile(asset)
		if err != nil {
			t.Fatalf("read %s: %v", asset, err)
		}
		source.Write(raw)
	}
	for _, required := range []string{
		"damageVignette", "flashCombatDamage",
		"abyssIntentChip", "enemy_intents",
		"abyssRoundAverage", "roundDurations",
		"ab-user-pinned", "logHtmlBtn", "new Blob",
		"hpForecast", "ab_consumable_order", "button.draggable=true",
		"ab_hud_order", "ab_threat_history", "abyssFloorAverage",
		"ab_boss_hp_mode", "abyssClearRating", "ab_boss_bookmarks",
		".ab-wavepips .ab-pip:nth-child", "abyssPetStatus", "data-loyalty",
		"ab-expiring", "Damage taken this fight", "bindLongPress",
		"Five-floor descent", "ab-slot-mismatch", "contextmenu",
		"ab_run_active_ms", "ab-mana-segmented", "Run depth ",
		"Cache split", "ab-low-hp",
	} {
		if !strings.Contains(source.String(), required) {
			t.Errorf("stage HUD is missing %q", required)
		}
	}
	if strings.Contains(source.String(), "probe.innerHTML") {
		t.Error("damage classification must not parse combat log markup through innerHTML")
	}
}
