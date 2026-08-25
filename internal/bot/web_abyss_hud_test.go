package bot

import (
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

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
	state := (&Bot{DB: db}).abyssHUDPageState("player", run, abyssStats{}, nil)
	if state.FloorsCleared != 2 || state.EscrowPerFloor != 1_200 {
		t.Fatalf("HUD pace = %d floors at %d/floor", state.FloorsCleared, state.EscrowPerFloor)
	}
	if state.Jackpot != 54_321 {
		t.Fatalf("jackpot = %d", state.Jackpot)
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
		"abyssJackpot", "bank ×", "Happy Hour −20%", "interestRatePct",
		"/api/abyss/focus", `aria-label="Reward focus"`,
	} {
		if !strings.Contains(source, required) {
			t.Errorf("HUD contract missing %q", required)
		}
	}
}
