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

func TestQuoteAbyssTransportBankKeepsEightyFivePercentAndClearsCache(t *testing.T) {
	t.Parallel()

	quote, ok := quoteAbyssTransportBank(1_001, 1.5)
	if !ok {
		t.Fatal("valid armored transport was rejected")
	}
	want := abyssPartialBankQuote{Escrow: 1_001, Gross: 1_501, Fee: 225, Payout: 1_276}
	if quote != want {
		t.Fatalf("transport quote = %#v, want %#v", quote, want)
	}
	if _, ok := quoteAbyssTransportBank(0, 2); ok {
		t.Fatal("empty cache accepted armored transport")
	}
}

func TestQuoteAbyssBankModeKeepsPartialAndTransportFeesExclusive(t *testing.T) {
	t.Parallel()

	partial, ok := quoteAbyssBankMode(1_000, 1.5, 25, false)
	if !ok || partial.PartialFee != 37 || partial.TransportFee != 0 || partial.Remaining != 750 {
		t.Fatalf("partial mode quote = %#v, %t", partial, ok)
	}
	transport, ok := quoteAbyssBankMode(1_000, 1.5, 0, true)
	if !ok || transport.PartialFee != 0 || transport.TransportFee != 225 || transport.Remaining != 0 {
		t.Fatalf("transport mode quote = %#v, %t", transport, ok)
	}
	if _, ok := quoteAbyssBankMode(1_000, 1.5, 25, true); ok {
		t.Fatal("combined partial and transport mode was accepted")
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

func TestPlanAbyssAutoInsuranceAppliesOrdinaryCoverOnlyToCompatibleRuns(t *testing.T) {
	t.Parallel()

	plan := planAbyssAutoInsurance(true, false, nil, 1_000, 0, 0)
	if plan != (abyssAutoInsurancePlan{Applied: true, Percent: 25, Cost: 125}) {
		t.Fatalf("auto-insurance plan = %#v", plan)
	}
	empty := planAbyssAutoInsurance(true, false, nil, 0, 0, 0)
	if !empty.Applied || empty.Percent != 25 || empty.Cost != 1 {
		t.Fatalf("empty-cache auto-insurance plan = %#v", empty)
	}
	for _, test := range []struct {
		name     string
		enabled  bool
		hardcore bool
		pacts    []string
	}{
		{name: "disabled"},
		{name: "hardcore", enabled: true, hardcore: true},
		{name: "uninsured", enabled: true, pacts: []string{"uninsured"}},
	} {
		got := planAbyssAutoInsurance(test.enabled, test.hardcore, test.pacts, 1_000, 0, 0)
		if got.Applied || got.Percent != 0 || got.Cost != 0 {
			t.Errorf("%s plan = %#v", test.name, got)
		}
	}
}

func TestAbyssOverkillGoldConversionAndCap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		damage     int
		floorBonus int64
		want       int64
	}{
		{name: "no excess", floorBonus: 1_000},
		{name: "reward-free floor", damage: 500},
		{name: "partial gold rounds up", damage: 1, floorBonus: 1_000, want: 1},
		{name: "ten damage per gold", damage: 100, floorBonus: 1_000, want: 10},
		{name: "quarter-floor cap", damage: 100_000, floorBonus: 1_000, want: 250},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := abyssOverkillGold(test.damage, test.floorBonus); got != test.want {
				t.Errorf("abyssOverkillGold(%d, %d) = %d, want %d", test.damage, test.floorBonus, got, test.want)
			}
		})
	}
}

func TestApplyAbyssEscrowRewardIncludesOverkillInSoftCap(t *testing.T) {
	t.Parallel()

	input := abyssEscrowRewardInput{
		Escrow:         200_000,
		FloorBonus:     1_000,
		Depth:          10,
		OverkillDamage: 100_000,
	}
	growth, credited := applyAbyssEscrowReward(input)
	if credited != 62 {
		t.Fatalf("soft-capped overkill credit = %d, want 62", credited)
	}
	if growth.Escrow != 200_312 || growth.Bonus != 312 {
		t.Fatalf("growth with overkill = %#v, want escrow 200312 and bonus 312", growth)
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
		`id="btnTransport" onclick="abyssBank(0,true)"`,
		`id="btnDoubleBonus"`,
		`/api/abyss/double_bonus`,
		`percent:Number(percent)||0`,
		`Partial-bank fee −10%`,
		`Armored transport fee −15%`,
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
		"armored transport requires a non-empty cache",
		"if !continuing && run.Depth > 0",
		"if !continuing {\n\t\ts.abyssOps.funnel.observeBank(uid)",
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
