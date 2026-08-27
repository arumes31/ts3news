package db

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestValidateAbyssEscrowSnapshot(t *testing.T) {
	t.Parallel()

	valid := testAbyssEscrowSnapshot(t)
	tests := []struct {
		name    string
		mutate  func(*AbyssEscrowSnapshot)
		wantErr string
	}{
		{name: "valid"},
		{name: "checksum mismatch", mutate: func(snapshot *AbyssEscrowSnapshot) { snapshot.Checksum = strings.Repeat("0", 64) }, wantErr: "checksum mismatch"},
		{name: "unsupported version", mutate: func(snapshot *AbyssEscrowSnapshot) { snapshot.Version++ }, wantErr: "unsupported"},
		{name: "orphan loot", mutate: func(snapshot *AbyssEscrowSnapshot) {
			snapshot.Loot = json.RawMessage(`[{"id":1,"client_uid":"missing","item_data":{}}]`)
			resealAbyssEscrowSnapshot(t, snapshot)
		}, wantErr: "has no active run"},
		{name: "missing session owner member", mutate: func(snapshot *AbyssEscrowSnapshot) {
			snapshot.Members = json.RawMessage(`[]`)
			resealAbyssEscrowSnapshot(t, snapshot)
		}, wantErr: "has no owner member"},
		{name: "non-object loot payload", mutate: func(snapshot *AbyssEscrowSnapshot) {
			snapshot.Loot = json.RawMessage(`[{"id":1,"client_uid":"player","item_data":[]}]`)
			resealAbyssEscrowSnapshot(t, snapshot)
		}, wantErr: "non-object item_data"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			snapshot := valid
			snapshot.Active = append(json.RawMessage(nil), valid.Active...)
			snapshot.Loot = append(json.RawMessage(nil), valid.Loot...)
			snapshot.Sessions = append(json.RawMessage(nil), valid.Sessions...)
			snapshot.Members = append(json.RawMessage(nil), valid.Members...)
			if test.mutate != nil {
				test.mutate(&snapshot)
			}
			err := ValidateAbyssEscrowSnapshot(snapshot)
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateAbyssEscrowSnapshot: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("ValidateAbyssEscrowSnapshot error = %v, want containing %q", err, test.wantErr)
			}
		})
	}
}

func TestExportAbyssEscrowSnapshot(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	mock.ExpectQuery(`FROM abyss_active`).WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte(`[]`)))
	mock.ExpectQuery(`FROM abyss_escrow_loot`).WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte(`[]`)))
	mock.ExpectQuery(`FROM abyss_combat_sessions`).WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte(`[]`)))
	mock.ExpectQuery(`FROM abyss_combat_members`).WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte(`[]`)))
	mock.ExpectCommit()

	now := time.Date(2026, time.August, 27, 8, 0, 0, 0, time.FixedZone("test", 2*60*60))
	snapshot, err := ExportAbyssEscrowSnapshot(context.Background(), database, now)
	if err != nil {
		t.Fatalf("ExportAbyssEscrowSnapshot: %v", err)
	}
	if snapshot.CreatedAt != now.UTC() {
		t.Fatalf("CreatedAt = %s, want %s", snapshot.CreatedAt, now.UTC())
	}
	if snapshot.Counts != (AbyssEscrowSnapshotCounts{}) {
		t.Fatalf("Counts = %+v, want empty", snapshot.Counts)
	}
	if err := ValidateAbyssEscrowSnapshot(snapshot); err != nil {
		t.Fatalf("exported snapshot is invalid: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func testAbyssEscrowSnapshot(t *testing.T) AbyssEscrowSnapshot {
	t.Helper()
	snapshot := AbyssEscrowSnapshot{
		Version:   AbyssEscrowSnapshotVersion,
		CreatedAt: time.Date(2026, time.August, 27, 6, 0, 0, 0, time.UTC),
		Active:    json.RawMessage(`[{"client_uid":"player"}]`),
		Loot:      json.RawMessage(`[{"id":1,"client_uid":"player","item_data":{"amount":5}}]`),
		Sessions:  json.RawMessage(`[{"session_id":"session","owner_uid":"player"}]`),
		Members:   json.RawMessage(`[{"session_id":"session","client_uid":"player"}]`),
	}
	resealAbyssEscrowSnapshot(t, &snapshot)
	return snapshot
}

func resealAbyssEscrowSnapshot(t *testing.T, snapshot *AbyssEscrowSnapshot) {
	t.Helper()
	if err := normalizeAbyssSnapshotTables(snapshot); err != nil {
		t.Fatal(err)
	}
	counts, err := countAbyssSnapshotRows(*snapshot)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Counts = counts
	snapshot.Checksum = abyssEscrowSnapshotChecksum(*snapshot)
}
