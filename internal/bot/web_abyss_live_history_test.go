package bot

import (
	"database/sql/driver"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

type abyssReplayJSONMatcher struct{}

func (abyssReplayJSONMatcher) Match(value driver.Value) bool {
	encoded, ok := value.(string)
	if !ok {
		return false
	}
	var archive abyssLiveReplayArchive
	if json.Unmarshal([]byte(encoded), &archive) != nil {
		return false
	}
	return archive.SessionID == "session" && archive.OwnerUID == "owner" &&
		archive.State.RandomSeed == [2]uint64{11, 22} &&
		len(archive.State.Events) == 1 && archive.State.Events[0].ID == 7
}

func TestAbyssLiveCombatEventsAfter(t *testing.T) {
	combat := &abyssLiveCombat{
		history: []abyssLiveEvent{
			{ID: 2, Snapshots: map[string]abyssLiveSnapshot{"user": {OK: true, Version: 2}}},
			{ID: 3, Snapshots: map[string]abyssLiveSnapshot{"user": {OK: true, Version: 3}}},
		},
	}

	events := combat.eventsAfter("user", 2)
	if len(events) != 1 || events[0].Version != 3 {
		t.Fatalf("eventsAfter(user, 2) = %+v, want version 3", events)
	}
	events[0].Version = 99
	if combat.history[1].Snapshots["user"].Version != 3 {
		t.Fatal("eventsAfter returned mutable internal history")
	}
}

func TestAbyssLiveCombatArchiveReplayPersistsEveryParticipant(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New(): %v", err)
	}
	defer func() { _ = db.Close() }()
	combat := &abyssLiveCombat{
		server:       &WebServer{bot: &Bot{DB: db}},
		id:           "session",
		ownerUID:     "owner",
		participants: map[string]bool{"owner": true, "helper": true},
		tactics:      map[string]string{},
		options:      map[string][]abyssLiveOption{},
		queued:       map[string]abyssLiveAction{},
		phase:        "complete",
		version:      7,
		randomSeed:   [2]uint64{11, 22},
		history: []abyssLiveEvent{{
			ID: 7, Phase: "complete", Snapshots: map[string]abyssLiveSnapshot{},
		}},
	}
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE abyss_combat_sessions").
		WithArgs("complete", 0, int64(7), nil, "", sqlmock.AnyArg(), "session").
		WillReturnResult(sqlmock.NewResult(0, 1))
	for _, uid := range []string{"helper", "owner"} {
		mock.ExpectExec("UPDATE abyss_combat_members").
			WithArgs("", "", nil, 0, sqlmock.AnyArg(), "session", uid).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs("abyss_live_replay_session_session", abyssReplayJSONMatcher{}).
		WillReturnResult(sqlmock.NewResult(0, 1))
	for _, uid := range []string{"helper", "owner"} {
		mock.ExpectExec("INSERT INTO app_meta").
			WithArgs("abyss_live_replay_user_"+uid+"_session", "session").
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	if err := combat.persist(); err != nil {
		t.Fatalf("persist(): %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestAbyssLiveCombatArchiveFailureRollsBackTerminalPersistence(t *testing.T) {
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
		tactics:      map[string]string{},
		options:      map[string][]abyssLiveOption{},
		queued:       map[string]abyssLiveAction{},
		phase:        "failed",
		version:      7,
		randomSeed:   [2]uint64{11, 22},
	}
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE abyss_combat_sessions").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE abyss_combat_members").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO app_meta").
		WillReturnError(driver.ErrBadConn)
	mock.ExpectRollback()

	if err := combat.persist(); err == nil {
		t.Fatal("persist() succeeded despite replay archive failure")
	}
	if len(combat.history) != 0 {
		t.Fatalf("history published after rollback: %+v", combat.history)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestHandleAbyssCombatEventsResumesAfterLastEventID(t *testing.T) {
	server := &WebServer{}
	combat := &abyssLiveCombat{
		id:           "session",
		participants: map[string]bool{"user": true},
		phase:        "complete",
		version:      3,
		history: []abyssLiveEvent{
			{ID: 1, At: time.Now(), Snapshots: map[string]abyssLiveSnapshot{"user": {OK: true, Version: 1, Phase: "planning"}}},
			{ID: 2, At: time.Now(), Snapshots: map[string]abyssLiveSnapshot{"user": {OK: true, Version: 2, Phase: "resolving"}}},
			{ID: 3, At: time.Now(), Snapshots: map[string]abyssLiveSnapshot{"user": {OK: true, Version: 3, Phase: "complete"}}},
		},
	}
	server.liveCombats.Store("session", combat)
	server.liveCombatByUID.Store("user", "session")

	request := httptest.NewRequest("GET", "/api/abyss/combat/events", nil)
	request.Header.Set("Last-Event-ID", "1")
	recorder := httptest.NewRecorder()
	server.handleAbyssCombatEvents(recorder, request, "user")

	body := recorder.Body.String()
	if strings.Contains(body, "id: 1\n") {
		t.Fatalf("resumed stream replayed acknowledged event:\n%s", body)
	}
	if !strings.Contains(body, "id: 2\n") || !strings.Contains(body, "id: 3\n") {
		t.Fatalf("resumed stream did not replay every later event:\n%s", body)
	}
	if strings.Index(body, "id: 2\n") > strings.Index(body, "id: 3\n") {
		t.Fatalf("resumed stream events are out of order:\n%s", body)
	}
}

func TestHandleAbyssCombatEventsRejectsInvalidLastEventID(t *testing.T) {
	server := &WebServer{}
	combat := &abyssLiveCombat{id: "session"}
	server.liveCombats.Store("session", combat)
	server.liveCombatByUID.Store("user", "session")

	request := httptest.NewRequest("GET", "/api/abyss/combat/events", nil)
	request.Header.Set("Last-Event-ID", "not-a-number")
	recorder := httptest.NewRecorder()
	server.handleAbyssCombatEvents(recorder, request, "user")

	if recorder.Code != 400 {
		t.Fatalf("invalid Last-Event-ID status = %d, want 400", recorder.Code)
	}
}

func TestHandleAbyssCombatEventsRejectsFutureLastEventID(t *testing.T) {
	server := &WebServer{}
	combat := &abyssLiveCombat{id: "session", version: 3}
	server.liveCombats.Store("session", combat)
	server.liveCombatByUID.Store("user", "session")

	request := httptest.NewRequest("GET", "/api/abyss/combat/events", nil)
	request.Header.Set("Last-Event-ID", "4")
	recorder := httptest.NewRecorder()
	server.handleAbyssCombatEvents(recorder, request, "user")

	if recorder.Code != 400 {
		t.Fatalf("future Last-Event-ID status = %d, want 400", recorder.Code)
	}
}
