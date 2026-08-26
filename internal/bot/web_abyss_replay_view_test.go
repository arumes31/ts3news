package bot

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestValidAbyssReplaySessionID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sessionID string
		want      bool
	}{
		{name: "uuid", sessionID: "01234567-89ab-cdef-0123-456789abcdef", want: true},
		{name: "underscore", sessionID: "test_session-4", want: true},
		{name: "empty"},
		{name: "separator", sessionID: "../../session"},
		{name: "whitespace", sessionID: "session one"},
		{name: "oversized", sessionID: strings.Repeat("x", abyssReplaySessionIDMax+1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := validAbyssReplaySessionID(test.sessionID); got != test.want {
				t.Errorf("validAbyssReplaySessionID(%q) = %v, want %v", test.sessionID, got, test.want)
			}
		})
	}
}

func TestDecodeAbyssReplayRequestRejectsUnboundedOrAmbiguousJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "unknown field", body: `{"session_id":"session","admin":true}`},
		{name: "multiple values", body: `{"session_id":"session"}{"view":true}`},
		{name: "oversized", body: `{"code":"` + strings.Repeat("x", abyssReplayRequestMaxBytes) + `"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := httptest.NewRequest(http.MethodPost, "/api/abyss/replay/code", strings.NewReader(test.body))
			if _, err := decodeAbyssReplayRequest(httptest.NewRecorder(), request); err == nil {
				t.Fatal("invalid replay request was accepted")
			}
		})
	}
}

func TestBuildAbyssReplayViewIsBoundedAndParticipantScoped(t *testing.T) {
	t.Parallel()

	events := make([]abyssLiveEvent, abyssReplayViewMaxFrames+5)
	for index := range events {
		logs := make([]string, abyssReplayViewMaxLogsPerFrame+1)
		for logIndex := range logs {
			logs[logIndex] = "combat line"
		}
		logs[len(logs)-1] = "&lt;img id=replay-xss&gt;<span>Victory</span>\x00" + strings.Repeat("x", abyssReplayViewTextMaxRunes)
		events[index] = abyssLiveEvent{
			ID: int64(index + 1), At: time.Unix(int64(index), 0).UTC(), Round: index + 1, Phase: "planning",
			Snapshots: map[string]abyssLiveSnapshot{
				"viewer": {
					Allies:     []abyssLiveCombatantView{{ID: "ally:private-uid", HP: 80, MaxHP: 100}},
					Enemies:    []abyssLiveCombatantView{{ID: "enemy:1", HP: 40, MaxHP: 100}},
					RecentLogs: logs,
				},
				"other": {RecentLogs: []string{"private other-player event"}},
			},
		}
	}
	archive := abyssLiveReplayArchive{
		SessionID: "session", OwnerUID: "owner-private-uid", ArchivedAt: time.Now().UTC(),
		State: abyssLivePersistedState{SchemaVersion: 1, RandomSeed: [2]uint64{11, 22}, Events: events},
	}

	view := buildAbyssReplayView(archive, "viewer")
	if !view.Truncated || view.TotalEvents != len(events) || len(view.Frames) != abyssReplayViewMaxFrames {
		t.Fatalf("view bounds = truncated %v, total %d, frames %d", view.Truncated, view.TotalEvents, len(view.Frames))
	}
	if view.Frames[0].EventID != 6 {
		t.Fatalf("first retained event = %d, want 6", view.Frames[0].EventID)
	}
	last := view.Frames[len(view.Frames)-1]
	if len(last.Logs) != abyssReplayViewMaxLogsPerFrame || last.Allies.HP != 80 || last.Enemies.HP != 40 {
		t.Fatalf("last frame = %+v", last)
	}
	if len([]rune(last.Logs[len(last.Logs)-1])) > abyssReplayViewTextMaxRunes || strings.ContainsRune(last.Logs[len(last.Logs)-1], '\x00') {
		t.Fatalf("log was not sanitized and bounded: %q", last.Logs[len(last.Logs)-1])
	}
	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal replay view: %v", err)
	}
	for _, privateValue := range []string{"owner-private-uid", "ally:private-uid", "private other-player event"} {
		if strings.Contains(string(encoded), privateValue) {
			t.Errorf("replay view leaked %q", privateValue)
		}
	}
}

func TestHandleAbyssReplayCodeReturnsOwnedReplayView(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	archive := abyssLiveReplayArchive{
		SessionID: "session-1", OwnerUID: "viewer", ArchivedAt: time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC),
		State: abyssLivePersistedState{
			SchemaVersion: 1, RandomSeed: [2]uint64{31, 41},
			Events: []abyssLiveEvent{{
				ID: 7, Round: 3, Phase: "complete",
				Snapshots: map[string]abyssLiveSnapshot{"viewer": {RecentLogs: []string{"victory"}}},
			}},
		},
	}
	raw, err := json.Marshal(archive)
	if err != nil {
		t.Fatalf("marshal archive: %v", err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM app_meta WHERE key=$1")).
		WithArgs("abyss_live_replay_user_viewer_session-1").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("session-1"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM app_meta WHERE key=$1")).
		WithArgs("abyss_live_replay_session_session-1").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(raw)))

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/replay/code", bytes.NewBufferString(
		`{"session_id":"session-1","view":true}`,
	))
	response := httptest.NewRecorder()
	server.handleAbyssReplayCode(response, request, "viewer")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		OK     bool            `json:"ok"`
		Replay abyssReplayView `json:"replay"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.OK || payload.Replay.RandomSeed != [2]uint64{31, 41} ||
		len(payload.Replay.Frames) != 1 || payload.Replay.Frames[0].Logs[0] != "victory" {
		t.Fatalf("unexpected replay response: %+v", payload)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAbyssReplayCodeRejectsMismatchedOwnership(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM app_meta WHERE key=$1")).
		WithArgs("abyss_live_replay_user_viewer_session-1").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("different-session"))

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/replay/code", bytes.NewBufferString(
		`{"session_id":"session-1","view":true}`,
	))
	response := httptest.NewRecorder()
	server.handleAbyssReplayCode(response, request, "viewer")
	if !strings.Contains(response.Body.String(), "replay not found") {
		t.Fatalf("response = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssReplayViewerAssetsAreSafelyWired(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read abyss.html: %v", err)
	}
	viewer, err := webAssets.ReadFile("webassets/abyss_replay.html")
	if err != nil {
		t.Fatalf("read abyss_replay.html: %v", err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_replay.css")
	if err != nil {
		t.Fatalf("read abyss_replay.css: %v", err)
	}
	root := abyssAAARepositoryRoot(t)
	server, err := os.ReadFile(filepath.Join(root, "internal", "bot", "web.go"))
	if err != nil {
		t.Fatalf("read web.go: %v", err)
	}
	for label, contract := range map[string]string{
		"stylesheet link": `{{asset "/static/abyss_replay.css"}}`,
		"viewer template": `{{template "abyssReplayViewerJS" .}}`,
	} {
		if !bytes.Contains(page, []byte(contract)) {
			t.Errorf("Abyss page is missing %s", label)
		}
	}
	if !bytes.Contains(server, []byte(`mux.HandleFunc("/static/abyss_replay.css"`)) {
		t.Error("production server does not register the replay stylesheet")
	}
	if !bytes.Contains(viewer, []byte("consEsc(String(line))")) {
		t.Error("replay log renderer does not escape server-provided lines")
	}
	if !bytes.Contains(styles, []byte(".ab-replay-frame")) {
		t.Error("replay frame styling is missing")
	}
}
