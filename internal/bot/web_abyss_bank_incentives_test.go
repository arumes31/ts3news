package bot

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssOvercapBankConversionRoundsWithoutDestroyingGold(t *testing.T) {
	t.Parallel()

	if got := abyssOvercapBankConversion(149_999, 10); got != (abyssOvercapConversion{}) {
		t.Fatalf("below-cap conversion = %#v, want none", got)
	}
	got := abyssOvercapBankConversion(2_150_999, 10)
	want := abyssOvercapConversion{Gold: 200_000, Tokens: 2}
	if got != want {
		t.Fatalf("over-cap conversion = %#v, want %#v", got, want)
	}
}

func TestAbyssBanksUntilFreeInsurance(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		streak int
		ready  bool
		want   int
	}{{0, false, 3}, {1, false, 2}, {2, false, 1}, {3, false, 3}, {8, true, 0}} {
		if got := abyssBanksUntilFreeInsurance(test.streak, test.ready); got != test.want {
			t.Errorf("banksUntil(%d, %t) = %d, want %d", test.streak, test.ready, got, test.want)
		}
	}
}

func TestConsumeAbyssFreeInsuranceUsesCallerTransaction(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	key := abyssFreeInsuranceKey("delver")
	mock.ExpectExec("INSERT INTO app_meta").WithArgs(key).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT value FROM app_meta").WithArgs(key).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("1"))
	mock.ExpectExec("UPDATE app_meta SET value='0'").WithArgs(key).
		WillReturnResult(sqlmock.NewResult(0, 1))
	used, err := consumeAbyssFreeInsurance(tx, "delver")
	if err != nil || !used {
		t.Fatalf("consume voucher = (%t, %v), want (true, nil)", used, err)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAwardAbyssBankStreakInsuranceOnlyOnThirdBank(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if earned, err := awardAbyssBankStreakInsurance(tx, "delver", 2); err != nil || earned {
		t.Fatalf("second bank award = (%t, %v), want (false, nil)", earned, err)
	}
	key := abyssFreeInsuranceKey("delver")
	mock.ExpectExec("INSERT INTO app_meta").WithArgs(key).WillReturnResult(sqlmock.NewResult(0, 1))
	if earned, err := awardAbyssBankStreakInsurance(tx, "delver", 3); err != nil || !earned {
		t.Fatalf("third bank award = (%t, %v), want (true, nil)", earned, err)
	}
	mock.ExpectCommit()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
