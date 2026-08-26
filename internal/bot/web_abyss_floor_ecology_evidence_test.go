package bot

import (
	"os"
	"slices"
	"strings"
	"testing"
)

func TestAbyssFloorEcologyRegistries(t *testing.T) {
	t.Parallel()

	events := map[string]string{
		"merchant":          "Abyssal Market",
		"wishing_well":      "Wishing Well",
		"puzzle":            "Three Chests",
		"cursed_library":    "Cursed Library",
		"den":               "Gambling Den",
		"rift":              "Scrying Rift",
		"blood_altar":       "Blood Altar",
		"alchemy_lab":       "Alchemy Lab",
		"mirrors":           "Hall of Mirrors",
		"locked_vault":      "Locked Vault",
		"collapsed_passage": "Collapsed Passage",
		"abyssal_garden":    "Abyssal Garden",
		"trap_chamber":      "Trap Chamber",
		"unstable_portal":   "Unstable Portal",
		"graveyard":         "Delver Graveyard",
		"echo_floor":        "Echo Floor",
		"bounty_board":      "Bounty Board",
		abyssForgeFloorType: "Silent Anvil",
		abyssEventChainType: "Triune Sigil Hunt",
	}
	for eventType, label := range events {
		raw := `{"type":"` + eventType + `"}`
		if got := abyssEventTypeLabel(raw); got != label {
			t.Errorf("event label %q = %q, want %q", eventType, got, label)
		}
	}
	for _, modifier := range []string{"mirror_clone", "storm_floor", "darkness"} {
		if !slices.Contains(abyssCombatFloorModifiers, modifier) {
			t.Errorf("combat room registry is missing %q", modifier)
		}
	}
}

func TestAbyssFloorEcologyResolutionContracts(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	eventUI, err := webAssets.ReadFile("webassets/abyss_events.html")
	if err != nil {
		t.Fatal(err)
	}
	server, err := os.ReadFile("web_abyss.go")
	if err != nil {
		t.Fatal(err)
	}
	rooms, err := os.ReadFile("web_abyss_rooms.go")
	if err != nil {
		t.Fatal(err)
	}
	traversal, err := os.ReadFile("web_abyss_traversal_rooms.go")
	if err != nil {
		t.Fatal(err)
	}
	joined := string(page) + string(eventUI) + string(server) + string(rooms) + string(traversal)
	for _, contract := range []string{
		"puzzle_pick", "trap_attempt", "vault_cache", "library_trade",
		"den_dice_high", "market_mystery", "abyssRiftPeek", "storm_floor",
		"darkness", "sanctuary_buy", "altar_sacrifice", "echo_claim",
		"lab_combine", "portal_enter", "graveyard_duel", "bounty_accept",
		"passage_squeeze", "well_toss", "garden_harvest", "mirrors_pick",
		"forge_floor_leave", "openAbyssForgeFloor", "forge_floor_used",
		"sigil_chain_accept", "renderAbyssEventChain", "chest_reward",
	} {
		if !strings.Contains(joined, contract) {
			t.Errorf("floor ecology is missing resolution contract %q", contract)
		}
	}
}
