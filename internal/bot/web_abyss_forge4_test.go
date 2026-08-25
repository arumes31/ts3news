package bot

import (
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestForge4PureMechanics(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		actions int
		want    int
	}{{-10, 0}, {49, 0}, {50, 1}, {249, 4}, {250, 5}, {10_000, 5}} {
		if got := forge4MasteryDiscountForCount(test.actions); got != test.want {
			t.Errorf("mastery discount at %d actions = %d, want %d", test.actions, got, test.want)
		}
	}
	if moved := forge4MasterworkTransferLevels(5, 0); moved != 4 {
		t.Errorf("masterwork transfer = %d, want 4", moved)
	}
	if moved := forge4MasterworkTransferLevels(5, 4); moved != 1 {
		t.Errorf("capped masterwork transfer = %d, want 1", moved)
	}
	base := content.Stats{HP: 10_000, STR: 2_000}
	masterworked := base
	for range 5 {
		masterworked = masterworked.Scaled(1.03)
	}
	restored := forge4RemoveMasterworkStats(masterworked, 5)
	if restored.HP > base.HP || restored.STR > base.STR {
		t.Errorf("masterwork removal retained duplicated stats: base=%+v restored=%+v", base, restored)
	}
	if count, critical := forge4CraftOutput(0.049999); count != 2 || !critical {
		t.Errorf("critical craft = %d, %t", count, critical)
	}
	if count, critical := forge4CraftOutput(0.05); count != 1 || critical {
		t.Errorf("craft boundary = %d, %t", count, critical)
	}
	if !abyssPerfectCorruption(0.049999) || abyssPerfectCorruption(0.05) {
		t.Error("perfect-corruption five-percent boundary is incorrect")
	}
	if forge4ReforgeDailyLimit(content.RarityLegendary) != 1 || forge4ReforgeDailyLimit(content.RarityEternal) != 2 {
		t.Error("Eternal reforge privilege does not grant exactly one additional daily use")
	}
	repairKit, ok := content.GetConsumableByID("repair_kit_ii")
	if !ok || repairKit.EffectValue != 50 {
		t.Errorf("Repair Kit II = %+v, %t; want 50 durability", repairKit, ok)
	}
}

func TestForge4GuidedAwakenOffersDistinctChoices(t *testing.T) {
	t.Parallel()

	for range 100 {
		options := forge4GuidedAwakenOptions()
		if len(options) != 3 {
			t.Fatalf("guided awaken returned %d choices", len(options))
		}
		seen := map[string]bool{}
		for _, option := range options {
			if option == "" || seen[option] {
				t.Fatalf("guided awaken choices are not distinct: %v", options)
			}
			seen[option] = true
		}
	}
}

func TestForge4RelocateGemPreservesEverySocket(t *testing.T) {
	t.Parallel()

	tests := []struct {
		from, to int
		want     []string
	}{{0, 2, []string{"B", "C", "A"}}, {2, 0, []string{"C", "A", "B"}}, {1, 2, []string{"A", "C", "B"}}}
	for _, test := range tests {
		original := []string{"A", "B", "C"}
		got, ok := forge4RelocateGem(original, test.from, test.to)
		if !ok || strings.Join(got, ",") != strings.Join(test.want, ",") {
			t.Errorf("relocate %d→%d = %v, %t; want %v", test.from, test.to, got, ok, test.want)
		}
		if strings.Join(original, ",") != "A,B,C" {
			t.Fatalf("relocation mutated source slice: %v", original)
		}
	}
	if _, ok := forge4RelocateGem([]string{"A"}, 0, 0); ok {
		t.Fatal("same-socket relocation was accepted")
	}
}

func TestForge4RoutesHavePlayerControls(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	source := string(page)
	for _, required := range []string{
		"/api/abyss/batch_temper", "/api/abyss/temper_guard", "/api/abyss/forge_queue",
		"/api/abyss/gem_upgrade_all", "/api/abyss/scrape_rune", "/api/abyss/unattune",
		"/api/abyss/masterwork_transfer", "/api/abyss/reforge_lock", "/api/abyss/rebalance_all",
		"/api/abyss/unbrand", "/api/abyss/special_reroll", "/api/abyss/awaken_guided",
		"/api/abyss/imbue_remove", "/api/abyss/polish_all", "/api/abyss/craft_repair_kit2",
		"/api/abyss/socket_relocate", "/api/abyss/fuse_preview", "/api/abyss/celestial_fuse_boosted",
		"/api/abyss/recipe_fav", "/api/abyss/convert_mats", "/api/abyss/sanctuary_undo2",
		"Blessed Celestial (50%)", "aria-pressed=\"{{.Favorite}}\"", "Quoted cost:",
		"forge4QuotedAction('/api/abyss/polish_all','polish_all'",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("Forge control contract is missing %q", required)
		}
	}
}
