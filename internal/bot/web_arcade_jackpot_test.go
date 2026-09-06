package bot

import (
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
			mock.ExpectQuery("SELECT amount FROM arcade_jackpots WHERE game_key=$1 FOR UPDATE").WithArgs("abyss").WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(int64(math.MaxInt64)))
			reset := mock.ExpectExec("UPDATE arcade_jackpots SET amount = 10000, updated_at = NOW() WHERE game_key=$1").WithArgs("abyss")
			if stage == "reset" {
				reset.WillReturnError(errors.New("reset failed"))
			} else {
				reset.WillReturnResult(sqlmock.NewResult(0, 1))
				credit := mock.ExpectExec("UPDATE users SET gold = LEAST(9223372036854775807::numeric, gold::numeric + $1)::bigint WHERE client_uid=$2").WithArgs(int64(math.MaxInt64), "winner")
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
