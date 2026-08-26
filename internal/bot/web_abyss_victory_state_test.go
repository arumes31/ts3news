package bot

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCommitAbyssVictoryRunStateCommitsEscrowAndFlagsTogether(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE abyss_active SET escrow").
		WithArgs(int64(9_000), "delver").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs(abyssRunFlagsKey("delver"), `{"event_chains":1}`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := commitAbyssVictoryRunState(database, "delver", 9_000, map[string]int64{
		abyssRunFlagEventChains: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCommitAbyssVictoryRunStateRollsBackBothOnFlagFailure(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE abyss_active SET escrow").
		WithArgs(int64(9_000), "delver").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs(abyssRunFlagsKey("delver"), sqlmock.AnyArg()).
		WillReturnError(errors.New("write failed"))
	mock.ExpectRollback()

	err = commitAbyssVictoryRunState(database, "delver", 9_000, map[string]int64{
		abyssRunFlagEventChains: 1,
	})
	if err == nil {
		t.Fatal("flag failure did not abort victory state")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
