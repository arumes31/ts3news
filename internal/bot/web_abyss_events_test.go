package bot

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestAbyssExpandedEventRules(t *testing.T) {
	t.Parallel()

	if got := abyssEventOffer(1_000, `{"type":"den","mem_mult":1.2}`); got != 1_200 {
		t.Fatalf("remembered event offer = %d, want 1200", got)
	}
	if got := abyssEventOffer(1_000, `{"type":"den","mem_mult":9}`); got != 1_500 {
		t.Fatalf("remembered event offer cap = %d, want 1500", got)
	}
	if _, available := abyssDenGameFor("den_dice_high", 40); available {
		t.Fatal("high-roller table unlocked at depth 40")
	}
	high, available := abyssDenGameFor("den_dice_high", 41)
	if !available || high.Stake != 3_000 || high.Prize != 6_000 || high.Odds != 0.5 {
		t.Fatalf("high-roller game = %+v, available %v", high, available)
	}

	prepared := prepareAbyssEventForDepth(`{"type":"merchant","items":[{"price":1000}]}`, 50)
	var state map[string]any
	if err := json.Unmarshal([]byte(prepared), &state); err != nil {
		t.Fatalf("decode prepared merchant: %v", err)
	}
	items, _ := state["items"].([]any)
	item, _ := items[0].(map[string]any)
	if state["depth"] != float64(50) || item["price"] != float64(1_500) {
		t.Fatalf("prepared merchant = %#v", state)
	}
	if state["mystery_available"] != true || state["mystery_price"] != float64(abyssMarketMysteryPrice) {
		t.Fatalf("merchant mystery slot = %#v", state)
	}
	if got := abyssAltarBuffDuration(`{"type":"blood_altar","mem_mult":1.2}`, false); got != 4 {
		t.Fatalf("remembered altar duration = %d, want 4", got)
	}
	if got := abyssAltarBuffDuration(`{"type":"blood_altar","mem_mult":1.2}`, true); got != 8 {
		t.Fatalf("corrupted altar duration = %d, want 8", got)
	}
	if got := abyssRiskyBrewDuration(3); got != 5 {
		t.Fatalf("risky brew duration = %d, want 5", got)
	}
	if abyssMirrorBuffDuration(2) != 3 || abyssMirrorBuffDuration(3) != 4 {
		t.Fatal("same-reflection empowerment threshold is invalid")
	}
	memory := advanceAbyssMirrorMemory(abyssMirrorMemory{}, "speed_elixir", "run-1")
	memory = advanceAbyssMirrorMemory(memory, "speed_elixir", "run-1")
	if memory.Streak != 1 {
		t.Fatalf("same run advanced mirror streak to %d", memory.Streak)
	}
	memory = advanceAbyssMirrorMemory(memory, "speed_elixir", "run-2")
	memory = advanceAbyssMirrorMemory(memory, "speed_elixir", "run-3")
	if memory.Streak != 3 {
		t.Fatalf("three-run mirror streak = %d, want 3", memory.Streak)
	}
}

func TestAbyssExpandedEventUIContracts(t *testing.T) {
	t.Parallel()

	source, err := webAssets.ReadFile("webassets/abyss_events.html")
	if err != nil {
		t.Fatal(err)
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	server, err := os.ReadFile("web_abyss.go")
	if err != nil {
		t.Fatal(err)
	}
	joined := string(source) + string(page) + string(server)
	for _, contract := range []string{
		"market_mystery", "den_dice_high",
		"library_trade_spd", "state.lifetime",
		"lab_risky", "corrupted sacrifice doubles", "mirror_streak",
		"55% Common · 28% Uncommon · 12% Rare · 4% Epic · 1% Legendary",
		"state.mem_mult",
	} {
		if !strings.Contains(joined, contract) {
			t.Errorf("expanded event UI is missing %q", contract)
		}
	}
}
