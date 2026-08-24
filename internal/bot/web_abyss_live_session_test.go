package bot

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestStartAbyssLiveCombatKeepsRegistryOnTransactionFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New(): %v", err)
	}
	defer func() { _ = db.Close() }()

	server := &WebServer{bot: &Bot{DB: db}}
	oldCombat := &abyssLiveCombat{id: "old-session"}
	server.liveCombats.Store("old-session", oldCombat)
	server.liveCombatByUID.Store("user", "old-session")

	mock.ExpectQuery("SELECT COALESCE\\(coop_uid, ''\\) FROM abyss_active").
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"coop_uid"}).AddRow(""))
	mock.ExpectQuery("SELECT value FROM app_meta WHERE key=\\$1").
		WithArgs("abyss_live_tactic_user").
		WillReturnError(errors.New("no tactic"))
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM abyss_combat_sessions").
		WithArgs("user").
		WillReturnError(errors.New("delete failed"))
	mock.ExpectRollback()

	if _, err := server.startAbyssLiveCombat("user", abyssRun{Depth: 7}, 8, abyssTier{}, "", ""); err == nil {
		t.Fatal("startAbyssLiveCombat() succeeded despite transaction failure")
	}
	if got, ok := server.liveCombatByUID.Load("user"); !ok || got != "old-session" {
		t.Fatalf("liveCombatByUID changed on transaction failure: %v, %t", got, ok)
	}
	if got, ok := server.liveCombats.Load("old-session"); !ok || got != oldCombat {
		t.Fatalf("liveCombats changed on transaction failure: %v, %t", got, ok)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestPersistedAbyssLiveSnapshotUsesAuthoritativeRecoveryValues(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New(): %v", err)
	}
	defer func() { _ = db.Close() }()

	server := &WebServer{bot: &Bot{DB: db}}
	mock.ExpectQuery("SELECT m.state::text, s.owner_uid, s.phase, s.session_id, s.depth").
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"state", "owner_uid", "phase", "session_id", "depth"}).
			AddRow(`{"session_id":"stale-session","version":4,"previous_depth":999}`, "owner", "planning", "authoritative-session", 7))
	mock.ExpectExec("UPDATE abyss_active SET depth=\\$1").
		WithArgs(7, "owner").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE abyss_combat_sessions SET phase='failed'").
		WithArgs(sqlmock.AnyArg(), int64(5), "authoritative-session").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE abyss_combat_members SET state=\\$1").
		WithArgs(sqlmock.AnyArg(), "authoritative-session", "user").
		WillReturnResult(sqlmock.NewResult(0, 1))

	snapshot, found := server.persistedAbyssLiveSnapshot("user")
	if !found || !snapshot.OK || snapshot.Phase != "failed" {
		t.Fatalf("persistedAbyssLiveSnapshot() = (%+v, %t), want failed snapshot", snapshot, found)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestPersistedAbyssLiveSnapshotDoesNotRestoreNonPositiveDepth(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New(): %v", err)
	}
	defer func() { _ = db.Close() }()

	server := &WebServer{bot: &Bot{DB: db}}
	mock.ExpectQuery("SELECT m.state::text, s.owner_uid, s.phase, s.session_id, s.depth").
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"state", "owner_uid", "phase", "session_id", "depth"}).
			AddRow(`{"session_id":"stale-session","version":4,"previous_depth":999}`, "owner", "planning", "authoritative-session", 0))
	mock.ExpectExec("UPDATE abyss_combat_sessions SET phase='failed'").
		WithArgs(sqlmock.AnyArg(), int64(5), "authoritative-session").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE abyss_combat_members SET state=\\$1").
		WithArgs(sqlmock.AnyArg(), "authoritative-session", "user").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if _, found := server.persistedAbyssLiveSnapshot("user"); !found {
		t.Fatal("persistedAbyssLiveSnapshot() did not return the persisted snapshot")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}
