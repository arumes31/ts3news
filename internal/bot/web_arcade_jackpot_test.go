package bot

import (
	"database/sql"
	"errors"
	"math"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestClaimJackpotBoundsCreditAndRollsBackErrors(t *testing.T) {
	t.Parallel()
	for _, stage := range []string{"success", "reset", "credit"} {
		t.Run(stage, func(t *testing.T) {
			t.Parallel()
			database, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT gold FROM users WHERE client_uid=$1 FOR UPDATE").WithArgs("winner").WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(0))
			mock.ExpectQuery("SELECT amount FROM arcade_jackpots WHERE game_key=$1 FOR UPDATE").WithArgs("abyss").WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(math.MaxInt64)))
			reset := mock.ExpectExec("UPDATE arcade_jackpots SET amount = amount - $1, updated_at = NOW() WHERE game_key=$2").WithArgs(int64(math.MaxInt64), "abyss")
			if stage == "reset" {
				reset.WillReturnError(errors.New("reset failed"))
			} else {
				reset.WillReturnResult(sqlmock.NewResult(0, 1))
				credit := mock.ExpectExec("/* economy:bot.Bot.claimJackpot */ UPDATE users SET gold = gold + $1 WHERE client_uid=$2").WithArgs(int64(math.MaxInt64), "winner")
				if stage == "credit" {
					credit.WillReturnError(errors.New("credit failed"))
				} else {
					credit.WillReturnResult(sqlmock.NewResult(0, 1))
				}
			}
			want := int64(0)
			if stage == "success" {
				mock.ExpectCommit()
				want = math.MaxInt64
			} else {
				mock.ExpectRollback()
			}
			bot := &Bot{DB: database}
			if got := bot.claimJackpot("winner", "abyss"); got != want {
				t.Fatalf("claim = %d, want %d", got, want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestClaimJackpotRetainsPoolBeyondWalletHeadroom(t *testing.T) {
	database, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gold FROM users WHERE client_uid=$1 FOR UPDATE").WithArgs("winner").WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(int64(math.MaxInt64 - 20)))
	mock.ExpectQuery("SELECT amount FROM arcade_jackpots WHERE game_key=$1 FOR UPDATE").WithArgs("global").WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(1000))
	mock.ExpectExec("UPDATE arcade_jackpots SET amount = amount - $1, updated_at = NOW() WHERE game_key=$2").WithArgs(int64(20), "global").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("/* economy:bot.Bot.claimJackpot */ UPDATE users SET gold = gold + $1 WHERE client_uid=$2").WithArgs(int64(20), "winner").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if got := (&Bot{DB: database}).claimJackpot("winner", "global"); got != 20 {
		t.Fatalf("credited%d instead of20", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestClaimJackpotMissingOrFullWalletCannotConsumePool(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(map[bool]string{false: "full", true: "missing"}[missing], func(t *testing.T) {
			database, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			mock.ExpectBegin()
			account := mock.ExpectQuery("SELECT gold FROM users WHERE client_uid=$1 FOR UPDATE").WithArgs("winner")
			if missing {
				account.WillReturnError(sql.ErrNoRows)
			} else {
				account.WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(int64(math.MaxInt64)))
				mock.ExpectQuery("SELECT amount FROM arcade_jackpots WHERE game_key=$1 FOR UPDATE").WithArgs("global").WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(1000))
			}
			mock.ExpectRollback()
			if got := (&Bot{DB: database}).claimJackpot("winner", "global"); got != 0 {
				t.Fatalf("invalid claim%d", got)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
