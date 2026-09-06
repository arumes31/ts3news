package bot

import (
	"errors"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssRaffleSettleBoundsCreditAndRollsBackFailure(t *testing.T) {
	t.Parallel()
	for _, fail := range []bool{false, true} {
		t.Run(strconv.FormatBool(fail), func(t *testing.T) {
			t.Parallel()
			database, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			day := abyssRaffleDay(time.Now().Add(-24 * time.Hour))
			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO app_meta (key, value) VALUES ($1, '1') ON CONFLICT (key) DO NOTHING`).WithArgs("abyss_raffle_settled_" + day).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectQuery("SELECT value FROM app_meta WHERE key=$1").WithArgs("abyss_raffle_pot_" + day).WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(strconv.FormatInt(math.MaxInt64, 10)))
			mock.ExpectQuery("SELECT key FROM app_meta WHERE key LIKE $1 ORDER BY key").WithArgs("abyss_raffle_entry_" + day + "_%").WillReturnRows(sqlmock.NewRows([]string{"key"}).AddRow("abyss_raffle_entry_" + day + "_winner"))
			credit := mock.ExpectExec("UPDATE users SET gold = LEAST(9223372036854775807::numeric, gold::numeric + $1)::bigint WHERE client_uid=$2").WithArgs(int64(math.MaxInt64), "winner")
			want := int64(math.MaxInt64)
			if fail {
				credit.WillReturnError(errors.New("credit failed"))
				mock.ExpectRollback()
				want = 0
			} else {
				credit.WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			}
			bot := &Bot{DB: database}
			if got := bot.abyssRaffleSettle("winner"); got != want {
				t.Fatalf("raffle = %d, want %d", got, want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
