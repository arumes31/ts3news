package bot

import (
	"math"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestAbyssRunFloorsCleared(t *testing.T) {
	tests := []struct {
		name string
		run  abyssRun
		want int
	}{
		{name: "surface", run: abyssRun{}, want: 0},
		{name: "cleared combat", run: abyssRun{Active: true, Depth: 3, FloorType: "combat"}, want: 3},
		{name: "checkpoint entry", run: abyssRun{Active: true, Depth: 20, CheckpointStart: 20, FloorType: "combat"}, want: 0},
		{name: "unresolved sanctuary", run: abyssRun{Active: true, Depth: 7, FloorType: "rest"}, want: 6},
		{name: "defeated floor", run: abyssRun{Active: true, Downed: true, Depth: 12, FloorType: "combat"}, want: 11},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := abyssRunFloorsCleared(test.run); got != test.want {
				t.Fatalf("floors cleared = %d, want %d", got, test.want)
			}
		})
	}
}

func TestAbyssHUDPageStateUsesAuthoritativeRunData(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT amount FROM arcade_jackpots").
		WithArgs("abyss").
		WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(54_321))
	mock.ExpectQuery("SELECT COALESCE\\(pacts, ''\\) FROM abyss_active").
		WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"pacts"}).AddRow("glass_cannon"))

	run := abyssRun{Active: true, Depth: 12, CheckpointStart: 10, Escrow: 2_400, FloorType: "combat"}
	equipped := map[content.GearSlot]content.Gear{
		content.SlotTrinket1: {ID: "ABYSS_LUCKY_COIN"},
	}
	state := (&Bot{DB: db}).abyssHUDPageState("player", run, abyssStats{UpInterest: 2}, equipped)
	if state.FloorsCleared != 2 || state.EscrowPerFloor != 1_200 {
		t.Fatalf("HUD pace = %d floors at %d/floor", state.FloorsCleared, state.EscrowPerFloor)
	}
	if state.Jackpot != 54_321 {
		t.Fatalf("jackpot = %d", state.Jackpot)
	}
	wantInterest := abyssGreedyInterestRate(abyssEffectiveInterest(2, true), run.Depth) * 100
	if math.Abs(state.InterestRatePct-wantInterest) > 0.0001 || state.InterestTotalPct <= state.InterestRatePct {
		t.Fatalf("interest = %.4f%% per floor, %.4f%% total", state.InterestRatePct, state.InterestTotalPct)
	}
	if len(state.Pacts) != 1 || state.Pacts[0].Key != "glass_cannon" {
		t.Fatalf("pacts = %#v", state.Pacts)
	}
	mock.ExpectQuery("SELECT slot, durability FROM user_gear").
		WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"slot", "durability"}).AddRow("Weapon", 37))
	if got := (&Bot{DB: db}).abyssEquippedDurability("player")["Weapon"]; got != 37 {
		t.Fatalf("weapon durability = %d", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssHUDAndFocusContracts(t *testing.T) {
	if got := abyssFocusPreference(map[string]int64{abyssRunFlagFocus: abyssFocusIDs["materials"]}); got != "materials" {
		t.Fatalf("focus = %q", got)
	}
	if got := abyssFocusPreference(map[string]int64{}); got != "" {
		t.Fatalf("automatic focus preference = %q", got)
	}
	if got := abyssFocusPreference(map[string]int64{abyssRunFlagFocus: 99}); got != "" {
		t.Fatalf("invalid focus preference = %q", got)
	}

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_ui200.css")
	if err != nil {
		t.Fatal(err)
	}
	live, err := webAssets.ReadFile("webassets/abyss_live.html")
	if err != nil {
		t.Fatal(err)
	}
	liveStyles, err := webAssets.ReadFile("webassets/abyss_live.css")
	if err != nil {
		t.Fatal(err)
	}
	source := string(page) + string(styles) + string(live) + string(liveStyles)
	for _, required := range []string{
		".HUD.FloorsCleared", "sessionTokensEarned", "∞ floors", "checkpoint in ", "mll.textContent = bankLocked",
		"calc(25% - 1px)", "UI-47: threat meter", "curDepth-lastRestDepth", "leaderDistance()",
		"bountyRingHTML()", "focusHUDHTML()", "activeRunPacts.forEach",
		"--cooldown-angle", ".kind-item.unavailable::before", "gearCondition()", "Last Stand ready", "Comeback +10% · this run",
		"abyssJackpot", "bank ×", "Happy Hour −20%", "Number('{{printf \"%.3f\" .HUD.InterestRatePct}}') || 0",
		"if(!Number.isFinite(interestRatePct))interestRatePct=0",
		"/api/abyss/focus", `aria-label="Reward focus"`,
	} {
		if !strings.Contains(source, required) {
			t.Errorf("HUD contract missing %q", required)
		}
	}

	uiCSS, err := webAssets.ReadFile("webassets/abyss_ui200.css")
	if err != nil {
		t.Fatal(err)
	}
	liveCSS, err := webAssets.ReadFile("webassets/abyss_live.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		".ab-hudchip.pact", ".ab-focus-chip select", "ab-jackpot-grow",
		".kind-item.unavailable::before", "conic-gradient",
	} {
		if !strings.Contains(string(uiCSS)+string(liveCSS), required) {
			t.Errorf("HUD styling contract missing %q", required)
		}
	}
}
