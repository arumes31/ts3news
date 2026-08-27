package bot

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestLoadAbyssSetupState(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	mock.ExpectQuery("SELECT amount FROM arcade_jackpots").WithArgs("abyss").WillReturnRows(
		sqlmock.NewRows([]string{"amount"}).AddRow(88_000),
	)
	mock.ExpectQuery("SELECT COALESCE\\(tier, 'normal'\\), COALESCE\\(MAX\\(depth\\), 0\\)").WithArgs("player").WillReturnRows(
		sqlmock.NewRows([]string{"tier", "depth"}).AddRow("normal", 42).AddRow("hell", 17),
	)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)").WithArgs("player").WillReturnRows(
		sqlmock.NewRows([]string{"runs", "wins", "deaths", "gold_banked", "best_depth"}).AddRow(3, 2, 1, 42_000, 21),
	)
	mock.ExpectQuery("SELECT abyss_free_entry_date IS NULL").WithArgs("player").WillReturnRows(
		sqlmock.NewRows([]string{"free"}).AddRow(true),
	)
	mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("abyss_entry_setup_player").WillReturnRows(
		sqlmock.NewRows([]string{"value"}).AddRow(`{"tier":"hell","pacts":["glass_cannon"],"kit":"arcanist","mutation":"piercing","loot_rule":"owner","focus":"loot"}`),
	)

	state, err := (&Bot{DB: db}).loadAbyssSetupState("player", 2_500)
	if err != nil {
		t.Fatal(err)
	}
	if state.TierBests["normal"] != 42 || state.TierBests["hell"] != 17 || state.TierBests["nightmare"] != 0 {
		t.Fatalf("tier bests = %#v", state.TierBests)
	}
	if state.Jackpot != 88_000 {
		t.Fatalf("jackpot = %d", state.Jackpot)
	}
	if state.LastSetup == nil || state.LastSetup.Tier != "hell" || state.LastSetup.Focus != "loot" {
		t.Fatalf("last setup = %#v", state.LastSetup)
	}
	if len(state.LastSetup.Pacts) != 1 || state.LastSetup.Pacts[0] != "glass_cannon" {
		t.Fatalf("last setup pacts = %#v", state.LastSetup.Pacts)
	}
	if !state.FreeEntryAvailable {
		t.Fatal("free entry should be available")
	}
	if state.Yesterday.Runs != 3 || state.Yesterday.Wins != 2 || state.Yesterday.Deaths != 1 || state.Yesterday.GoldBanked != 42_000 || state.Yesterday.BestDepth != 21 {
		t.Fatalf("yesterday = %#v", state.Yesterday)
	}
	tier, _ := abyssTierByKey("normal")
	if state.FloorOneRiskByTier["normal"] != abyssRiskPct(1, tier, 2_500) {
		t.Fatalf("normal risk = %d", state.FloorOneRiskByTier["normal"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCanonicalAbyssEntrySetup(t *testing.T) {
	setup := canonicalAbyssEntrySetup(abyssEntrySetup{
		Tier: "forged", Pacts: []string{"glass_cannon", "forged", "glass_cannon"},
		Start: "checkpoint", Checkpoint: 13, Kit: "forged", Mutation: "forged",
		Position: "forged", LootRule: "forged", VeteranTrack: "forged", Focus: "forged",
	})
	if setup.Tier != "normal" || setup.Start != "" || setup.Checkpoint != 0 {
		t.Fatalf("canonical route = %#v", setup)
	}
	if setup.Kit != "vanguard" || setup.Position != "frontline" || setup.Mutation != "empowered" || setup.LootRule != "owner" || setup.VeteranTrack != "" || setup.Focus != "auto" {
		t.Fatalf("canonical options = %#v", setup)
	}
	if len(setup.Pacts) != 1 || setup.Pacts[0] != "glass_cannon" {
		t.Fatalf("canonical pacts = %#v", setup.Pacts)
	}
}

func TestCanonicalAbyssStorySetupOwnsTheRoute(t *testing.T) {
	setup := canonicalAbyssEntrySetup(abyssEntrySetup{
		Tier: "normal", StoryCampaign: true, Expedition: true,
		Start: "checkpoint", Checkpoint: 20,
	})
	if !setup.StoryCampaign || setup.Expedition || setup.Start != "" || setup.Checkpoint != 0 {
		t.Fatalf("story setup = %#v", setup)
	}
}

func TestSaveAbyssEntrySetup(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO app_meta (key, value) VALUES ($1, $2)")).
		WithArgs("abyss_entry_setup_player", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	if err := saveAbyssEntrySetup(db, "player", abyssEntrySetup{Tier: "hell", Focus: "tokens"}); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
