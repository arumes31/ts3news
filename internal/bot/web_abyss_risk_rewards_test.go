package bot

import (
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestQuoteAbyssPartialBankAppliesExactShareMultiplierAndFee(t *testing.T) {
	t.Parallel()

	quote, ok := quoteAbyssPartialBank(1_001, 1.5, 25)
	if !ok {
		t.Fatal("valid 25% partial bank was rejected")
	}
	want := abyssPartialBankQuote{Escrow: 250, Gross: 375, Fee: 37, Payout: 338, Remaining: 751}
	if quote != want {
		t.Fatalf("quote = %#v, want %#v", quote, want)
	}
	for _, percent := range []int{0, 10, 75, 100} {
		if _, ok := quoteAbyssPartialBank(1_000, 1.0, percent); ok {
			t.Errorf("unsupported %d%% partial bank was accepted", percent)
		}
	}
}

func TestResolveAbyssDoubleBonusAddsOrRemovesOnlyTheFloorBonus(t *testing.T) {
	t.Parallel()

	if got := resolveAbyssDoubleBonus(1_000, 200, true); got != 1_200 {
		t.Fatalf("winning escrow = %d, want 1200", got)
	}
	if got := resolveAbyssDoubleBonus(1_000, 200, false); got != 800 {
		t.Fatalf("losing escrow = %d, want 800", got)
	}
	if got := resolveAbyssDoubleBonus(100, 200, false); got != 0 {
		t.Fatalf("losing escrow underflowed to %d", got)
	}
	flags := map[string]int64{abyssRunFlagDoubleBonus: 200, abyssRunFlagDoubleBonusDepth: 8}
	if got := pendingAbyssDoubleBonus(flags, 8); got != 200 {
		t.Fatalf("same-depth pending bonus = %d, want 200", got)
	}
	if got := pendingAbyssDoubleBonus(flags, 9); got != 0 {
		t.Fatalf("stale pending bonus = %d, want 0", got)
	}
}

func TestAbyssGraceAndHardcoreForfeitPoliciesAreMutuallyExclusive(t *testing.T) {
	t.Parallel()

	grace := planAbyssForfeit(1_000, 0, 3, false)
	if grace != (abyssForfeitPolicy{Refund: 1_000, PreserveLoot: true}) {
		t.Fatalf("grace policy = %#v", grace)
	}
	normal := planAbyssForfeit(1_000, 50, 4, false)
	if normal != (abyssForfeitPolicy{Refund: 500, CountDeath: true}) {
		t.Fatalf("normal policy = %#v", normal)
	}
	hardcore := planAbyssForfeit(1_000, 75, 2, true)
	if hardcore != (abyssForfeitPolicy{CountDeath: true}) {
		t.Fatalf("hardcore policy = %#v", hardcore)
	}
	if got := abyssHardcoreFloorReward(250, true); got != 500 {
		t.Fatalf("hardcore floor reward = %d, want 500", got)
	}
}

func TestAbyssHardcoreLeaderboardUsesDedicatedRuns(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectQuery("WHERE a.tier = \\$1 AND a.hardcore = TRUE").
		WithArgs("normal", 10).
		WillReturnRows(sqlmock.NewRows([]string{"client_uid", "nick", "depth", "gold"}).
			AddRow("iron-uid", "Iron", 22, int64(900)))
	bot := &Bot{DB: database}
	rows := bot.topHardcoreDescents("normal", 10)
	if len(rows) != 1 || rows[0].Nickname != "Iron" || rows[0].Depth != 22 || rows[0].Rank != 1 {
		t.Fatalf("hardcore leaderboard = %#v", rows)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestConsumePendingAbyssDoubleBonusPreservesOtherRunFlags(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	tx, err := database.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs(abyssRunFlagsKey("player"), `{"double_bonus":0,"double_bonus_depth":0,"other":7}`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	flags := map[string]int64{abyssRunFlagDoubleBonus: 200, abyssRunFlagDoubleBonusDepth: 8, "other": 7}
	if err := consumePendingAbyssDoubleBonus(tx, "player", flags); err != nil {
		t.Fatalf("consume pending bonus: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if flags["other"] != 7 || flags[abyssRunFlagDoubleBonus] != 0 || flags[abyssRunFlagDoubleBonusDepth] != 0 {
		t.Fatalf("consumed flags = %#v", flags)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssRiskRewardControlsExposeOnlySupportedActions(t *testing.T) {
	t.Parallel()

	page, err := os.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	source := string(page)
	for _, required := range []string{
		"Begin a Hardcore run?",
		"Enter Hardcore",
		`id="btnBank25" onclick="abyssBank(25)"`,
		`id="btnBank50" onclick="abyssBank(50)"`,
		`id="btnDoubleBonus"`,
		`/api/abyss/double_bonus`,
		`percent:Number(percent)||0`,
		`Partial-bank fee −10%`,
		`pendingDoubleBonus=0`,
		`id="hardcoreMode"`,
		`hardcore:!!(hardcore&&hardcore.checked)`,
		`Grace floors return the full cache and preserve its loot.`,
	} {
		if !strings.Contains(source, required) {
			t.Errorf("Abyss risk/reward controls are missing %q", required)
		}
	}
	server, err := os.ReadFile("web_abyss.go")
	if err != nil {
		t.Fatalf("read Abyss handlers: %v", err)
	}
	serverSource := string(server)
	for _, required := range []string{
		"partial bank must be 25% or 50%",
		"resolve the floor-bonus gamble or descend before partial banking",
		"if !partial && run.Depth > 0",
		"if !partial {\n\t\ts.abyssOps.funnel.observeBank(uid)",
	} {
		if !strings.Contains(serverSource, required) {
			t.Errorf("Abyss risk/reward handlers are missing %q", required)
		}
	}
	migration, err := os.ReadFile("../db/migrations/0072_abyss_hardcore.up.sql")
	if err != nil {
		t.Fatalf("read hardcore migration: %v", err)
	}
	for _, required := range []string{"ADD COLUMN IF NOT EXISTS hardcore", "WHERE hardcore = TRUE"} {
		if !strings.Contains(string(migration), required) {
			t.Errorf("hardcore migration is missing %q", required)
		}
	}
}
