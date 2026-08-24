package bot

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAbyssWeeklyExpeditionIsStableForISOWeek(t *testing.T) {
	t.Parallel()

	seedA, ruleA := abyssWeeklyExpedition(time.Date(2026, time.August, 24, 1, 0, 0, 0, time.UTC))
	seedB, ruleB := abyssWeeklyExpedition(time.Date(2026, time.August, 30, 23, 0, 0, 0, time.UTC))
	seedNext, _ := abyssWeeklyExpedition(time.Date(2026, time.August, 31, 1, 0, 0, 0, time.UTC))
	if seedA != seedB || ruleA != ruleB {
		t.Fatalf("same ISO week changed expedition: (%d, %#v) != (%d, %#v)", seedA, ruleA, seedB, ruleB)
	}
	if seedNext == seedA {
		t.Fatalf("next ISO week retained seed %d", seedA)
	}
	flags := map[string]int64{abyssRunFlagWeeklySeed: seedA}
	stored, ok := abyssWeeklyRuleFromFlags(flags)
	if !ok || stored != ruleA {
		t.Fatalf("stored seed resolved to (%#v, %v), want %#v", stored, ok, ruleA)
	}
}

func TestPublicFloorCandidatesDoNotLeakHiddenRoomTypes(t *testing.T) {
	t.Parallel()

	candidates := []floorCandidate{
		{Index: 0, Type: "combat", Label: "Press onward", Icon: "swords"},
		{Index: 1, Type: "rest", Label: "Rest", Icon: "sanctuary"},
	}
	hidden := publicFloorCandidates(candidates, false)
	encoded, err := json.Marshal(hidden)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"combat", "rest", "Press onward", "sanctuary"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("hidden route response leaked %q: %s", secret, encoded)
		}
	}
	if hidden[0].Index != 0 || hidden[1].Index != 1 {
		t.Fatalf("opaque choices lost authoritative indexes: %#v", hidden)
	}

	revealed := publicFloorCandidates(candidates, true)
	if !revealed[0].Revealed || revealed[0].Label != candidates[0].Label || revealed[1].Revealed {
		t.Fatalf("cartographer reveal was not limited to one route: %#v", revealed)
	}
}

func TestAbyssSpecialRoomRollsCoverRunStructureRooms(t *testing.T) {
	t.Parallel()

	rolls := []float64{0.00, 0.04, 0.08, 0.12, 0.16}
	want := []string{"challenge_room", "cursed_door", "story_crossroads", "lost_explorer", "locked_vault"}
	for i, roll := range rolls {
		if got := abyssSpecialRoomForRoll(roll); !strings.Contains(got, want[i]) {
			t.Fatalf("roll %.2f produced %q, want %q", roll, got, want[i])
		}
	}
	if got := abyssSpecialRoomForRoll(0.20); got != "" {
		t.Fatalf("ordinary event roll produced special room %q", got)
	}
}

func TestPrepareAbyssEventForDepthScalesMerchantPrices(t *testing.T) {
	t.Parallel()

	raw := `{"type":"merchant","items":[{"name":"Potion","price":100}]}`
	var shallow, deep struct {
		Depth int `json:"depth"`
		Items []struct {
			Price int64 `json:"price"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(prepareAbyssEventForDepth(raw, 10)), &shallow); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(prepareAbyssEventForDepth(raw, 80)), &deep); err != nil {
		t.Fatal(err)
	}
	if shallow.Depth != 10 || deep.Depth != 80 || shallow.Items[0].Price != 110 || deep.Items[0].Price != 180 {
		t.Fatalf("unexpected merchant scaling: shallow=%#v deep=%#v", shallow, deep)
	}
}
