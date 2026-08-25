package bot

import (
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRecordAbyssAffixRunUsesCallerTransaction(t *testing.T) {
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
	mock.ExpectExec("INSERT INTO abyss_affix_stats").
		WithArgs("player", "bloodlust", 1, 37).
		WillReturnResult(sqlmock.NewResult(1, 1))
	if err := recordAbyssAffixRun(tx, "player", "bloodlust", 37, true); err != nil {
		t.Fatal(err)
	}
	mock.ExpectCommit()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAbyssAffixStatsCalculatesRates(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	mock.ExpectQuery("SELECT affix_key, runs, wins, total_depth FROM abyss_affix_stats").
		WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"affix_key", "runs", "wins", "total_depth"}).
			AddRow("bloodlust", 4, 3, 50))
	stats, err := (&Bot{DB: db}).loadAbyssAffixStats("player")
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != len(abyssDailyMods) {
		t.Fatalf("stats = %d, want %d", len(stats), len(abyssDailyMods))
	}
	for _, stat := range stats {
		if stat.Key == "bloodlust" && (stat.Runs != 4 || stat.Wins != 3 || stat.WinRatePct != 75 || stat.AverageDepth != 12.5) {
			t.Fatalf("bloodlust stats = %#v", stat)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssAffixStatsOutcomeAndClientContract(t *testing.T) {
	t.Parallel()

	bank, err := os.ReadFile("web_abyss.go")
	if err != nil {
		t.Fatal(err)
	}
	forfeit, err := os.ReadFile("web_abyss_econ.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bank), "recordAbyssAffixRun(tx") || !strings.Contains(string(forfeit), "recordAbyssAffixRun(tx") {
		t.Fatal("bank and forfeit do not both record authoritative affix outcomes")
	}
	module, err := webAssets.ReadFile("webassets/abyss_pact_program.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"affixPersonalStats", "win_rate_pct", "average_depth"} {
		if !strings.Contains(string(module), required) {
			t.Errorf("affix stats UI is missing %q", required)
		}
	}
}
