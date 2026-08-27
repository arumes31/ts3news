package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCheckAbyssEscrowIntegrity(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectQuery(`SELECT[\s\S]*FROM abyss_active`).WillReturnRows(
		sqlmock.NewRows([]string{"active", "loot", "sessions", "members", "orphan_loot", "orphan_sessions", "sessions_without_owner", "bad_payloads", "orphan_members"}).
			AddRow(2, 5, 1, 2, 0, 0, 0, 0, 0),
	)
	report, err := CheckAbyssEscrowIntegrity(context.Background(), database)
	if err != nil {
		t.Fatalf("CheckAbyssEscrowIntegrity: %v", err)
	}
	if !report.Healthy() {
		t.Fatalf("healthy report marked unhealthy: %+v", report)
	}
	if report.Counts != (AbyssEscrowSnapshotCounts{Active: 2, Loot: 5, Sessions: 1, Members: 2}) {
		t.Fatalf("Counts = %+v", report.Counts)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDrillAbyssEscrowRestoreRollsBack(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	for _, table := range []string{"abyss_active", "abyss_escrow_loot", "abyss_combat_sessions", "abyss_combat_members"} {
		mock.ExpectExec(`CREATE TEMP TABLE ` + table + `_restore_drill`).WillReturnResult(sqlmock.NewResult(0, 0))
	}
	for _, table := range []string{"abyss_active", "abyss_escrow_loot", "abyss_combat_sessions", "abyss_combat_members"} {
		mock.ExpectExec(`INSERT INTO ` + table + `_restore_drill`).WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 0))
	}
	for _, table := range []string{"abyss_active", "abyss_escrow_loot", "abyss_combat_sessions", "abyss_combat_members"} {
		mock.ExpectQuery(`FROM ` + table + `_restore_drill`).WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte(`[]`)))
	}
	mock.ExpectRollback()

	snapshot := AbyssEscrowSnapshot{Version: AbyssEscrowSnapshotVersion, CreatedAt: time.Now().UTC()}
	if err := sealAbyssEscrowSnapshot(&snapshot); err != nil {
		t.Fatal(err)
	}
	counts, err := DrillAbyssEscrowRestore(context.Background(), database, snapshot)
	if err != nil {
		t.Fatalf("DrillAbyssEscrowRestore: %v", err)
	}
	if counts != (AbyssEscrowSnapshotCounts{}) {
		t.Fatalf("Counts = %+v, want empty", counts)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDrillAbyssEscrowRestoreRejectsInvalidSnapshotBeforeDatabase(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	_, err = DrillAbyssEscrowRestore(context.Background(), database, AbyssEscrowSnapshot{})
	if err == nil {
		t.Fatal("DrillAbyssEscrowRestore accepted an invalid snapshot")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCheckAbyssEscrowIntegrityPropagatesQueryFailure(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("read failed"))
	if _, err := CheckAbyssEscrowIntegrity(context.Background(), database); err == nil {
		t.Fatal("CheckAbyssEscrowIntegrity returned nil error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
