package bot

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSplitAbyssJackpotBoundsCreditAndRollsBackFailure(t *testing.T) {
	t.Parallel()
	for _, fail := range []bool{false, true} {
		t.Run(strconv.FormatBool(fail), func(t *testing.T) {
			t.Parallel()
			database, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			split := int64(math.MaxInt64 / 10)
			mock.ExpectBegin()
			mock.ExpectExec("UPDATE users SET gold=gold-$1 WHERE client_uid=$2 AND gold >= $1").WithArgs(split, "winner").WillReturnResult(sqlmock.NewResult(0, 1))
			credit := mock.ExpectExec("UPDATE users SET gold=LEAST(9223372036854775807::numeric, gold::numeric+$1)::bigint WHERE client_uid=$2").WithArgs(split, "helper")
			want := split
			if fail {
				credit.WillReturnError(errors.New("credit failed"))
				mock.ExpectRollback()
				want = 0
			} else {
				credit.WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`INSERT INTO abyss_economy_events (client_uid,kind,message,amount) VALUES ($1,'jackpot_split',$2,$3)`).WithArgs("helper", fmt.Sprintf("Co-op jackpot share received: %dg.", split), split).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			}
			bot := &Bot{DB: database}
			if got := bot.splitAbyssJackpot("winner", "helper", math.MaxInt64); got != want {
				t.Fatalf("split = %d, want %d", got, want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
