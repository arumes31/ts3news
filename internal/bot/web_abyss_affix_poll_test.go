package bot

import (
	"database/sql"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssWeekendAffixOptionsAndWindow(t *testing.T) {
	t.Parallel()

	monday := time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC)
	options := abyssWeekendAffixOptions(monday)
	if len(options) != 3 || options[0] == options[1] || options[1] == options[2] || options[0] == options[2] {
		t.Fatalf("weekend options = %v", options)
	}
	if !abyssWeekendVotingOpen(monday) || abyssWeekendVotingOpen(monday.AddDate(0, 0, 5)) {
		t.Fatal("weekend vote window does not close at Saturday UTC")
	}
	if got := abyssWeekendStartsAt(monday).Format("2006-01-02"); got != "2026-08-29" {
		t.Fatalf("weekend starts = %s", got)
	}
}

func TestAbyssWeekendAffixPollTalliesAndSelectsDeterministically(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	at := time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC)
	options := abyssWeekendAffixOptions(at)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value, COUNT(*) FROM app_meta WHERE key LIKE $1 GROUP BY value")).
		WithArgs("abyss_weekend_affix_vote_2026-W35_%").
		WillReturnRows(sqlmock.NewRows([]string{"value", "count"}).AddRow(options[0], 4).AddRow(options[1], 7))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM app_meta WHERE key=$1")).
		WithArgs("abyss_weekend_affix_vote_2026-W35_player").
		WillReturnError(sql.ErrNoRows)
	status, err := (&Bot{DB: db}).abyssWeekendAffixPoll("player", at)
	if err != nil {
		t.Fatal(err)
	}
	if status.Winner != options[1] || status.Options[1].Votes != 7 || !status.VotingOpen {
		t.Fatalf("poll status = %#v", status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCurrentDailyChallengeUsesLockedWeekendWinner(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	saturday := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	options := abyssWeekendAffixOptions(saturday)
	mock.ExpectQuery("SELECT value, COUNT").
		WithArgs("abyss_weekend_affix_vote_2026-W35_%").
		WillReturnRows(sqlmock.NewRows([]string{"value", "count"}).AddRow(options[2], 9))
	_, got := (&Bot{DB: db}).currentDailyChallengeAt(saturday)
	if got != options[2] {
		t.Fatalf("weekend affix = %q, want winner %q", got, options[2])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssWeekendAffixPollClientContract(t *testing.T) {
	t.Parallel()

	module, err := webAssets.ReadFile("webassets/abyss_pact_program.html")
	if err != nil {
		t.Fatal(err)
	}
	text := string(module)
	for _, required := range []string{"affixWeekendPoll", "voteAbyssWeekendAffix", "/api/abyss/affix/weekend_vote", "aria-pressed"} {
		if !strings.Contains(text, required) {
			t.Errorf("weekend affix poll UI is missing %q", required)
		}
	}
	if strings.Contains(text, "innerHTML") {
		t.Error("weekend affix poll must render labels with text nodes")
	}
}
