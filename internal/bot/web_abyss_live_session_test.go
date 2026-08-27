package bot

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssLiveCombatPersistIsAtomic(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New(): %v", err)
	}
	defer func() { _ = db.Close() }()
	mock.MatchExpectationsInOrder(false)

	combat := &abyssLiveCombat{
		server:        &WebServer{bot: &Bot{DB: db}},
		id:            "session",
		ownerUID:      "owner",
		participants:  map[string]bool{"owner": true, "helper": true},
		tactics:       map[string]string{"owner": "balanced", "helper": "defensive"},
		phase:         "planning",
		round:         2,
		version:       4,
		options:       map[string][]abyssLiveOption{},
		queued:        map[string]abyssLiveAction{},
		idempotency:   map[string]abyssLiveIdempotency{},
		previousDepth: 7,
	}

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE abyss_combat_sessions").
		WithArgs("planning", 2, int64(4), nil, "", sqlmock.AnyArg(), "session").
		WillReturnResult(sqlmock.NewResult(0, 1))
	for _, uid := range []string{"owner", "helper"} {
		mock.ExpectExec("UPDATE abyss_combat_members").
			WithArgs(combat.tactics[uid], "", nil, 0, sqlmock.AnyArg(), "session", uid).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	if err := combat.persist(); err != nil {
		t.Fatalf("persist(): %v", err)
	}
	if len(combat.history) != 1 || combat.history[0].ID != 4 {
		t.Fatalf("persisted history = %+v, want event 4", combat.history)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestAbyssLiveCombatPersistRollsBackWholeSnapshot(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New(): %v", err)
	}
	defer func() { _ = db.Close() }()

	combat := &abyssLiveCombat{
		server:        &WebServer{bot: &Bot{DB: db}},
		id:            "session",
		ownerUID:      "owner",
		participants:  map[string]bool{"owner": true},
		tactics:       map[string]string{"owner": "balanced"},
		phase:         "planning",
		round:         2,
		version:       4,
		options:       map[string][]abyssLiveOption{},
		queued:        map[string]abyssLiveAction{},
		idempotency:   map[string]abyssLiveIdempotency{},
		previousDepth: 7,
	}

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE abyss_combat_sessions").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE abyss_combat_members").
		WillReturnError(errors.New("member write failed"))
	mock.ExpectRollback()

	if err := combat.persist(); err == nil {
		t.Fatal("persist() succeeded despite member write failure")
	}
	if len(combat.history) != 0 {
		t.Fatalf("failed persistence published %d history events", len(combat.history))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestAbyssLiveCombatPersistDoesNotPublishFailedCommit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New(): %v", err)
	}
	defer func() { _ = db.Close() }()

	combat := &abyssLiveCombat{
		server:       &WebServer{bot: &Bot{DB: db}},
		id:           "session",
		ownerUID:     "owner",
		participants: map[string]bool{"owner": true},
		tactics:      map[string]string{"owner": "balanced"},
		phase:        "planning",
		round:        2,
		version:      4,
		options:      map[string][]abyssLiveOption{},
		queued:       map[string]abyssLiveAction{},
		idempotency:  map[string]abyssLiveIdempotency{},
	}

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE abyss_combat_sessions").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE abyss_combat_members").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	if err := combat.persist(); err == nil {
		t.Fatal("persist() succeeded despite commit failure")
	}
	if len(combat.history) != 0 {
		t.Fatalf("failed commit published %d history events", len(combat.history))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestAbyssLiveCombatCapturePersistenceCreatesInitialEvent(t *testing.T) {
	combat := &abyssLiveCombat{
		id:           "session",
		ownerUID:     "owner",
		participants: map[string]bool{"owner": true, "helper": true},
		tactics:      map[string]string{"owner": "balanced", "helper": "defensive"},
		phase:        "starting",
		options:      map[string][]abyssLiveOption{},
		queued:       map[string]abyssLiveAction{},
		idempotency:  map[string]abyssLiveIdempotency{},
		randomSeed:   [2]uint64{31, 41},
	}
	_ = combat.IntN(100)

	owner, history, members, err := combat.capturePersistence()
	if err != nil {
		t.Fatalf("capturePersistence(): %v", err)
	}
	if owner.SchemaVersion != abyssLiveSnapshotSchemaVersion {
		t.Fatalf("initial schema version = %d, want %d", owner.SchemaVersion, abyssLiveSnapshotSchemaVersion)
	}
	if owner.RandomSeed != [2]uint64{31, 41} || owner.RandomDraws != 1 {
		t.Fatalf("initial replay cursor = seed %v draws %d, want [31 41] and 1", owner.RandomSeed, owner.RandomDraws)
	}
	if len(history) != 1 || history[0].ID != 0 || len(history[0].Snapshots) != 2 {
		t.Fatalf("initial history = %+v, want event 0 for both members", history)
	}
	if len(members) != 2 || members[0].uid != "helper" || members[1].uid != "owner" {
		t.Fatalf("initial members = %+v, want deterministic helper/owner order", members)
	}
}

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

	provenance, err := json.Marshal(abyssRunProvenance{
		Version: abyssRunProvenanceVersion,
		Seed:    [2]uint64{11, 22},
		Choices: []abyssRunChoice{},
		Floors:  []abyssRunFloorRecord{},
	})
	if err != nil {
		t.Fatalf("marshal provenance: %v", err)
	}
	mock.ExpectQuery("SELECT value FROM app_meta WHERE key=\\$1").
		WithArgs(abyssRunProvenanceKey("user")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(provenance)))
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
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT m.state::text, s.state::text, s.owner_uid, s.phase, s.session_id, s.depth").
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"member_state", "session_state", "owner_uid", "phase", "session_id", "depth"}).
			AddRow(
				`{"session_id":"stale-session","version":4,"previous_depth":999,"random_seed":[11,22]}`,
				`{"schema_version":1,"random_seed":[11,22],"snapshot":{"session_id":"stale-session"},"events":[]}`,
				"owner", "planning", "authoritative-session", 7,
			))
	mock.ExpectExec("UPDATE abyss_active SET depth=\\$1").
		WithArgs(7, "owner").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE abyss_combat_sessions SET phase='failed'").
		WithArgs(sqlmock.AnyArg(), int64(5), "authoritative-session").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE abyss_combat_members SET state=\\$1").
		WithArgs(sqlmock.AnyArg(), "authoritative-session").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT client_uid FROM abyss_combat_members").
		WithArgs("authoritative-session").
		WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("user"))
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs("abyss_live_replay_session_authoritative-session", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs("abyss_live_replay_user_user_authoritative-session", "authoritative-session").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	snapshot, found := server.persistedAbyssLiveSnapshot("user")
	if !found || !snapshot.OK || snapshot.Phase != "failed" {
		t.Fatalf("persistedAbyssLiveSnapshot() = (%+v, %t), want failed snapshot", snapshot, found)
	}
	if snapshot.SessionID != "authoritative-session" || snapshot.PreviousDepth != 7 {
		t.Fatalf("authoritative snapshot values = session %q depth %d", snapshot.SessionID, snapshot.PreviousDepth)
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
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT m.state::text, s.state::text, s.owner_uid, s.phase, s.session_id, s.depth").
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"member_state", "session_state", "owner_uid", "phase", "session_id", "depth"}).
			AddRow(
				`{"session_id":"stale-session","version":4,"previous_depth":999}`,
				`{"schema_version":1,"snapshot":{"session_id":"stale-session"},"events":[]}`,
				"owner", "planning", "authoritative-session", 0,
			))
	mock.ExpectExec("UPDATE abyss_combat_sessions SET phase='failed'").
		WithArgs(sqlmock.AnyArg(), int64(5), "authoritative-session").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE abyss_combat_members SET state=\\$1").
		WithArgs(sqlmock.AnyArg(), "authoritative-session").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT client_uid FROM abyss_combat_members").
		WithArgs("authoritative-session").
		WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("user"))
	mock.ExpectExec("INSERT INTO app_meta").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO app_meta").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if _, found := server.persistedAbyssLiveSnapshot("user"); !found {
		t.Fatal("persistedAbyssLiveSnapshot() did not return the persisted snapshot")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}
