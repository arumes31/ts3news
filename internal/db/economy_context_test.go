package db

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSetEconomyContext(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	mock.ExpectBegin()
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("SELECT set_config").WithArgs("arcade.coinflip", "request-1", "run-1", "item-1").WillReturnResult(sqlmock.NewResult(0, 1))
	if err := SetEconomyContext(context.Background(), tx, "arcade.coinflip", "request-1", "run-1", "item-1"); err != nil {
		t.Fatal(err)
	}
	mock.ExpectRollback()
	_ = tx.Rollback()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSetEconomyContextRejectsInvalidMetadataAndReportsDatabaseFailure(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	mock.ExpectBegin()
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"", strings.Repeat("x", 129), "bad\x00source"} {
		if err := SetEconomyContext(context.Background(), tx, value, "", "", ""); err == nil {
			t.Fatalf("accepted source %q", value)
		}
	}
	want := errors.New("database offline")
	mock.ExpectExec("SELECT set_config").WillReturnError(want)
	if err := SetEconomyContext(context.Background(), tx, "forge.temper", "", "", ""); !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
	mock.ExpectRollback()
	_ = tx.Rollback()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
