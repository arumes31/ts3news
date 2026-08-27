package bot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssCompetitionPeriodCanonicalization(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		period     string
		wantPeriod string
		build      string
		wantBuild  string
	}{
		{name: "season defaults", period: "unknown", wantPeriod: "season", build: "", wantBuild: ""},
		{name: "weekly warden", period: "weekly", wantPeriod: "weekly", build: "warden", wantBuild: "warden"},
		{name: "all time rejects unknown build", period: "all_time", wantPeriod: "all_time", build: "mage", wantBuild: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := canonicalAbyssCompetitionPeriod(test.period); got != test.wantPeriod {
				t.Fatalf("period = %q, want %q", got, test.wantPeriod)
			}
			if got := canonicalAbyssCompetitionBuild(test.build); got != test.wantBuild {
				t.Fatalf("build = %q, want %q", got, test.wantBuild)
			}
		})
	}
}

func TestAbyssCompetitionWeekUsesISOMondayBoundary(t *testing.T) {
	t.Parallel()
	key, start, end := abyssCompetitionWeekAt(time.Date(2026, time.August, 27, 18, 0, 0, 0, time.UTC))
	if key != "2026-W35" || start.Weekday() != time.Monday || end.Sub(start) != 7*24*time.Hour {
		t.Fatalf("week = %s %s..%s", key, start, end)
	}
}

func TestNewAbyssCompetitionRunRecordCreatesVerifiableChainHash(t *testing.T) {
	t.Parallel()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectQuery("SELECT COALESCE").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"build"}).AddRow("warden"))
	mock.ExpectQuery("SELECT channel_id").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"channel_id"}).AddRow(42))
	mock.ExpectQuery("SELECT audit_hash").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"audit_hash"}).AddRow("previous"))
	mock.ExpectQuery("SELECT value FROM app_meta").WithArgs(abyssRunProvenanceKey("delver")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}))
	run := abyssRun{Depth: 27, Tier: "hell", StartedAt: time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)}
	record, err := (&Bot{DB: database}).newAbyssCompetitionRunRecord("delver", run, 12345, true, false, "banked", 1.75)
	if err != nil {
		t.Fatal(err)
	}
	if record.Build != "warden" || !record.ChannelID.Valid || record.ChannelID.Int64 != 42 || len(record.AuditHash) != 64 {
		t.Fatalf("record = %#v", record)
	}
	var audit abyssCompetitionAudit
	if err := json.Unmarshal([]byte(record.AuditJSON), &audit); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(audit)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	if got := hex.EncodeToString(digest[:]); got != record.AuditHash || audit.PreviousHash != "previous" {
		t.Fatalf("audit chain mismatch: hash=%s previous=%q", got, audit.PreviousHash)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAbyssWagerJoinIsAtomic(t *testing.T) {
	t.Parallel()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gold FROM users").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(5000))
	mock.ExpectExec("INSERT INTO abyss_wager_entries").WithArgs(sqlmock.AnyArg(), 1000, "delver").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("UPDATE users SET gold=gold-\\$1").WithArgs(1000, "delver").WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(4000))
	mock.ExpectCommit()
	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest("POST", "/api/abyss/competition/wager/join", strings.NewReader(`{"bracket":1000}`))
	response := httptest.NewRecorder()
	server.handleAbyssWagerJoin(response, request, "delver")
	if !strings.Contains(response.Body.String(), `"ok":true`) || !strings.Contains(response.Body.String(), `"gold":4000`) {
		t.Fatalf("response = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssCompetitionProgramContracts(t *testing.T) {
	t.Parallel()
	root := abyssAAARepositoryRoot(t)
	checks := map[string][]string{
		filepath.Join(root, "internal", "db", "migrations", "0101_abyss_competition_program.up.sql"): {
			"audit_hash", "abyss_competition_snapshots", "abyss_competition_rewards", "abyss_wager_entries", "shame_opt_in",
		},
		filepath.Join(root, "internal", "bot", "abyss_competition_settlement.go"): {
			"season_badge_", "season_trophy_", "You were passed by", "settleAbyssWagerWeek",
		},
		filepath.Join(root, "internal", "bot", "webassets", "abyss_competition.html"): {
			"Abyss Competition", "Jump to me", "Past season snapshots", "Personal depth · last 30 days", "Opt into Hall of Shame",
		},
		filepath.Join(root, "internal", "bot", "web_abyss_competition.go"): {
			"Depth 20 speedruns", "Weekly vault", "Pact survival", "Bestiary families", "Hall of shame", "Bank streaks", "Companion power",
		},
		filepath.Join(root, "internal", "bot", "abyss_competition_channel.go"): {
			"Channel Abyss · this week", "sanitizeBBCode",
		},
	}
	for path, required := range checks {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, token := range required {
			if !strings.Contains(string(raw), token) {
				t.Errorf("%s missing %q", filepath.Base(path), token)
			}
		}
	}
}
