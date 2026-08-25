package bot

import (
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"ts3news/internal/content"
)

func TestAbyssCoreRiskRewardMath(t *testing.T) {
	t.Parallel()

	if got := abyssGreedyInterestRate(0.05, 29); got != 0.09 {
		t.Fatalf("greedy interest = %v, want 0.09", got)
	}
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	if got := abyssIdleDangerPct(now.Add(-90*time.Minute), now); got != 50 {
		t.Fatalf("idle danger cap = %d, want 50", got)
	}
	if got := abyssFranticBankFee(1_000_000, 149, 1_000); got != 50_000 {
		t.Fatalf("frantic fee = %d, want 50000", got)
	}
	if got := abyssFranticBankFee(1_000_000, 150, 1_000); got != 0 {
		t.Fatalf("fee at threshold = %d, want 0", got)
	}
	for _, valid := range []int{10, 15, 50, 85, 90} {
		if !abyssInsurancePercentValid(valid) {
			t.Errorf("insurance rejected valid percentage %d", valid)
		}
	}
	for _, invalid := range []int{0, 9, 12, 91, 100} {
		if abyssInsurancePercentValid(invalid) {
			t.Errorf("insurance accepted invalid percentage %d", invalid)
		}
	}
	if !abyssCheapskateEligible(5, 101) || abyssCheapskateEligible(5, 100) {
		t.Fatal("cheapskate eligibility does not enforce a strict 5% premium boundary")
	}
	removed, tokens := abyssRestCacheConversion(1_000_000)
	if removed != 500_000 || tokens != 3 {
		t.Fatalf("cache conversion = (%d, %d), want (500000, 3)", removed, tokens)
	}
	if got := abyssDeathWishFloorReward(400, true); got != 800 {
		t.Fatalf("death-wish reward = %d, want 800", got)
	}
	if got := abyssAnchorRefund(100, 1_000, true); got != 500 {
		t.Fatalf("anchor refund = %d, want 500", got)
	}
	if got := abyssEchoBankSeed(1_000_000, false); got != 50_000 {
		t.Fatalf("echo seed = %d, want 50000", got)
	}
	if got := abyssEchoBankSeed(1_000_000, true); got != 100_000 {
		t.Fatalf("double echo seed = %d, want 100000", got)
	}
	if got := abyssReviveOfferChancePct(3, 2); got != 76 {
		t.Fatalf("revive offer chance = %d, want 76", got)
	}
	if abyssColdMusclesOnEntry(0) != 0 || abyssColdMusclesOnEntry(10) != 2 {
		t.Fatal("cold-muscles entry duration does not match the two-floor route rule")
	}
	if got := abyssRecordPushReward(1_000, 11, 10); got != 1_030 {
		t.Fatalf("record-push reward = %d, want 1030", got)
	}
	if got := abyssRecordPushReward(1_000, 10, 10); got != 1_000 {
		t.Fatalf("non-record reward = %d, want 1000", got)
	}
	if got := abyssNextDefensiveMomentum(9, true); got != 10 {
		t.Fatalf("defensive momentum = %d, want 10", got)
	}
	if got := abyssNextDefensiveMomentum(8, false); got != 0 {
		t.Fatalf("damaged defensive momentum = %d, want 0", got)
	}
	if got := abyssFatigueDamage(10_000, 30); got != 0 {
		t.Fatalf("round-30 fatigue = %d, want 0", got)
	}
	if got := abyssFatigueDamage(10_000, 31); got != 100 {
		t.Fatalf("round-31 fatigue = %d, want 100", got)
	}
	normal, ok := abyssTierByKey("normal")
	if !ok || !abyssHybridSurge(true, 5) || abyssHybridSurge(true, 4) {
		t.Fatal("hybrid surge cadence is invalid")
	}
	next, ok := abyssNextTier(normal.Key)
	if !ok {
		t.Fatal("normal tier has no hybrid successor")
	}
	if got, want := abyssHybridDangerMultiplier(normal), next.DiffMult/normal.DiffMult; math.Abs(got-want) > 1e-9 {
		t.Fatalf("hybrid danger multiplier = %v, want %v", got, want)
	}
	if got, want := abyssHybridRewardMultiplier(normal), (normal.RewardMult+next.RewardMult)/2/normal.RewardMult; math.Abs(got-want) > 1e-9 {
		t.Fatalf("hybrid reward multiplier = %v, want %v", got, want)
	}
	user := UserInCombat{killerExp: map[string]int{string(content.MobCommon): 10}}
	if got := abyssKillerDamage(1_000, &user, &content.Mob{Type: content.MobCommon}); got != 1_010 {
		t.Fatalf("matching killer-family damage = %d, want 1010", got)
	}
	if got := abyssKillerDamage(1_000, &user, &content.Mob{Type: content.MobElite}); got != 1_000 {
		t.Fatalf("unmatched killer-family damage = %d, want 1000", got)
	}
}

func TestAbyssRaffleEntryCommitsWithBankTransaction(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs("abyss_raffle_pot_2026-08-25", "125", int64(125)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs("abyss_raffle_entry_2026-08-25_uid-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := recordAbyssRaffleEntry(tx, "uid-1", 125, time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("record raffle entry: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssCoreRiskGuards(t *testing.T) {
	t.Parallel()

	if abyssBankNeedsSafeWord(1_000_000, true) {
		t.Fatal("safe word should apply only above one million")
	}
	if !abyssBankNeedsSafeWord(1_000_001, true) {
		t.Fatal("safe word missing above one million")
	}
	if abyssBankNeedsSafeWord(2_000_000, false) {
		t.Fatal("disabled safe-word preference was ignored")
	}
	if got := normalizeAbyssSafeWord("  bank\t"); got != "BANK" {
		t.Fatalf("normalized safe word = %q, want BANK", got)
	}
	dayOne := time.Date(2026, 8, 25, 23, 59, 0, 0, time.UTC)
	dayTwo := dayOne.Add(2 * time.Minute)
	if abyssReviveStreakKeyAt("uid", dayOne) == abyssReviveStreakKeyAt("uid", dayTwo) {
		t.Fatal("daily revive streak key did not rotate at UTC midnight")
	}
	run := abyssRun{Depth: 20}
	base := abyssLastStandCost(run.Depth)
	if cost, available := abyssLastStandOffer(run, nil); cost != base || !available {
		t.Fatalf("first Last Stand = (%d, %v), want (%d, true)", cost, available, base)
	}
	run.LastStandUsed = true
	if cost, available := abyssLastStandOffer(run, map[string]int64{}); cost != base*3 || !available {
		t.Fatalf("second Last Stand = (%d, %v), want (%d, true)", cost, available, base*3)
	}
	if _, available := abyssLastStandOffer(run, map[string]int64{abyssRunFlagSecondLastStandUsed: 1}); available {
		t.Fatal("third Last Stand was offered")
	}
}

func TestAbyssCoreRiskAssetsAndTabs(t *testing.T) {
	t.Parallel()

	module, err := webAssets.ReadFile("webassets/abyss_core_risk.html")
	if err != nil {
		t.Fatalf("read risk module: %v", err)
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	navigation, err := webAssets.ReadFile("webassets/abyss_navigation.html")
	if err != nil {
		t.Fatalf("read navigation module: %v", err)
	}
	joined := string(module) + string(page) + string(navigation)
	for _, contract := range []string{
		`min="{{.CoreLoop.InsuranceMin}}"`,
		`/api/abyss/downed_timeout`,
		`/api/abyss/rest_cache_shrink`,
		`bankSafeWordToggle`,
		`deathWishToggle`,
		`anchorRuneBtn`,
		`abyssInterestMarker`,
		`updateAbyssReviveRisk`,
		`id="hybridMode"`,
		`data-abyss-section="shop"`,
		`data-abyss-section="forge"`,
		`{key:'shop',label:'🜲 Shop'}`,
		`{key:'forge',label:'⚒️ Forge'}`,
	} {
		if !strings.Contains(joined, contract) {
			t.Errorf("Abyss core UI is missing %q", contract)
		}
	}
	server, err := os.ReadFile("web_abyss.go")
	if err != nil {
		t.Fatalf("read Abyss server: %v", err)
	}
	if !strings.Contains(string(server), "DamageTaken: combatUsers[0].DamageTaken") {
		t.Fatal("perfect-run tracking does not read the mutated combatant")
	}
	combat, err := os.ReadFile("xp.go")
	if err != nil {
		t.Fatalf("read combat engine: %v", err)
	}
	if got := strings.Count(string(combat), "DamageTaken +="); got < 5 {
		t.Fatalf("perfect-run tracking covers %d damage paths, want at least 5", got)
	}
	if !strings.Contains(string(combat), "abyssFatigueDamage(u.Stats.HP, r)") ||
		!strings.Contains(string(combat), "abyssFatigueDamage(mob.MaxHP, r)") {
		t.Fatal("round fatigue is not wired into combat")
	}
}
