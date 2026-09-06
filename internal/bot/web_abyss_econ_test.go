package bot

import (
	"errors"
	"math"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssCapTaxPreservesExactLargePayout(t *testing.T) {
	t.Parallel()
	const wantTax int64 = 7_378_697_629_483_820_645
	after, tax := abyssCapTax(math.MaxInt64, 0)
	if tax != wantTax || after != math.MaxInt64-wantTax {
		t.Fatalf("tax split = (%d, %d), want (%d, %d)", after, tax, math.MaxInt64-wantTax, wantTax)
	}
}

func TestTaxAbyssDayGoldPropagatesCounterFailure(t *testing.T) {
	t.Parallel()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	failure := errors.New("counter unavailable")
	mock.ExpectExec("UPDATE users SET abyss_day = CURRENT_DATE").WithArgs("delver").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT abyss_day_gold FROM users WHERE client_uid=\\$1 FOR UPDATE").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"abyss_day_gold"}).AddRow(int64(abyssDayGoldCap)))
	mock.ExpectExec("UPDATE users SET abyss_day_gold = LEAST").WithArgs(int64(100), "delver").WillReturnError(failure)
	bot := &Bot{DB: database}
	after, tax, err := bot.taxAbyssDayGold(database, "delver", 100)
	if !errors.Is(err, failure) || after != 0 || tax != 0 {
		t.Fatalf("failed tax = (%d, %d, %v)", after, tax, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTaxAbyssDayGoldUsesBoundedDatabaseAccumulators(t *testing.T) {
	t.Parallel()
	database, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	const wantTax int64 = 7_378_697_629_483_820_645
	mock.ExpectExec(`UPDATE users SET abyss_day = CURRENT_DATE, abyss_day_gold = 0
 WHERE client_uid=$1 AND (abyss_day IS NULL OR abyss_day < CURRENT_DATE)`).WithArgs("delver").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT abyss_day_gold FROM users WHERE client_uid=$1 FOR UPDATE").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"abyss_day_gold"}).AddRow(int64(math.MaxInt64)))
	mock.ExpectExec("UPDATE users SET abyss_day_gold = LEAST(9223372036854775807::numeric, abyss_day_gold::numeric + $1)::bigint WHERE client_uid=$2").WithArgs(int64(math.MaxInt64), "delver").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE arcade_jackpots SET amount = LEAST(9223372036854775807::numeric, amount::numeric + $1)::bigint, updated_at = NOW() WHERE game_key='abyss'").WithArgs(wantTax).WillReturnResult(sqlmock.NewResult(0, 1))
	bot := &Bot{DB: database}
	after, tax, err := bot.taxAbyssDayGold(database, "delver", math.MaxInt64)
	if err != nil || tax != wantTax || after != math.MaxInt64-wantTax {
		t.Fatalf("tax split = (%d, %d, %v)", after, tax, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
