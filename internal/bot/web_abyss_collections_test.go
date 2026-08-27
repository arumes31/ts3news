package bot

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestAbyssBiomeMasteryAppliesAfterFiftyClears(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	biome := content.AbyssBiomes()[0]
	mock.ExpectQuery("SELECT value FROM app_meta").
		WithArgs(abyssBiomeMasteryKey("delver", biome.Name)).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("50"))
	got, mastered := (&Bot{DB: database}).applyAbyssBiomeMastery(
		"delver", biome, content.Stats{HP: 100, STR: 50, DEF: 25, SPD: 10, INT: 5},
	)
	if !mastered || got.HP != 102 || got.STR != 51 || got.DEF != 25 {
		t.Fatalf("mastery stats = %+v, mastered %v", got, mastered)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssSetBookMilestonesShareCanonicalBonus(t *testing.T) {
	total := abyssNamedSetTotal()
	if total < 4 {
		t.Fatalf("named set catalog unexpectedly small: %d", total)
	}
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectQuery("SELECT value FROM app_meta").
		WithArgs(abyssSetBookCountKey("collector")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(total))
	bonus := content.TreeBonus{Pct: map[string]float64{}}
	(&Bot{DB: database}).applyAbyssSetBookBonuses("collector", &bonus)
	if bonus.Pct["loot_find"] != 0.02 || bonus.Pct["gold_find"] != 0.03 || bonus.Pct["material_yield"] != 0.05 {
		t.Fatalf("set-book bonus = %#v", bonus.Pct)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGrantAbyssLoreCompletionAwardsTitleAndBadge(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectExec("INSERT INTO abyss_lore_unlocked").
		WithArgs("reader", len(abyssLoreFragments)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM abyss_lore_unlocked WHERE client_uid=$1")).
		WithArgs("reader").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(len(abyssLoreFragments)))
	mock.ExpectExec("INSERT INTO abyss_achievements").
		WithArgs("reader", abyssLoreAchievement).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs(abyssLoreTitleKey("reader"), abyssLoreTitle).
		WillReturnResult(sqlmock.NewResult(0, 1))
	unlocked, tokens, err := grantAbyssLoreFragment(database, "reader", len(abyssLoreFragments))
	if err != nil || !unlocked || tokens != 0 {
		t.Fatalf("lore completion = unlocked %v tokens %d err %v", unlocked, tokens, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBadgeCombinationUsesBothSlots(t *testing.T) {
	got := abyssBadgeCombination("depth_10", abyssLoreAchievement)
	if got != "Threshold Breaker (Depth 10) · Abyss Chronicler (Lore Complete)" {
		t.Fatalf("badge combination = %q", got)
	}
}
