package bot

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestLoadAbyssRunReplayVerifiesScopesAndSanitizesAudit(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	seed := [2]uint64{18_446_744_073_709_551_600, 9_223_372_036_854_775_809}
	audit := abyssCompetitionAudit{
		Version: 1, UID: "private-user", Depth: 8, Victory: true, Tier: "hell",
		StartedAt: "2026-08-27T10:00:00Z", EndedAt: "2026-08-27T10:10:00Z", EndReason: "banked",
		RunSeed: &seed,
		Choices: []abyssRunChoice{{Depth: 8, Kind: "<b>floor</b>", Value: "combat"}},
		Floors: []abyssRunFloorRecord{{
			Depth: 8, Biome: "<img src=x>Vault", Victory: true, HP: 90, MaxHP: 100,
			Seed: seed, Logs: []string{"<script>bad()</script> Victory"},
		}},
	}
	raw, err := json.Marshal(audit)
	if err != nil {
		t.Fatalf("marshal audit: %v", err)
	}
	digest := sha256.Sum256(raw)
	hash := hex.EncodeToString(digest[:])
	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT audit_hash,audit_data FROM abyss_runs WHERE id=$1 AND client_uid=$2",
	)).WithArgs(int64(44), "viewer").
		WillReturnRows(sqlmock.NewRows([]string{"audit_hash", "audit_data"}).AddRow(hash, raw))

	server := &WebServer{bot: &Bot{DB: database}}
	view, err := server.loadAbyssRunReplay(t.Context(), "viewer", 44)
	if err != nil {
		t.Fatalf("loadAbyssRunReplay: %v", err)
	}
	if view.RunID != 44 || len(view.Floors) != 1 || len(view.RunSeed) != 2 {
		t.Fatalf("replay view = %+v", view)
	}
	if view.RunSeed[0] != "18446744073709551600" || view.Floors[0].Seed[1] != "9223372036854775809" {
		t.Fatalf("seed strings lost precision: run=%v floor=%v", view.RunSeed, view.Floors[0].Seed)
	}
	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal replay view: %v", err)
	}
	for _, forbidden := range []string{"private-user", "<img", "<script>"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Errorf("replay view leaked %q: %s", forbidden, encoded)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAbyssRunReplayRejectsUnownedRun(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	mock.ExpectQuery("SELECT audit_hash,audit_data").WithArgs(int64(7), "viewer").
		WillReturnError(sql.ErrNoRows)

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodGet, "/api/abyss/run/replay?id=7", nil)
	response := httptest.NewRecorder()
	server.handleAbyssRunReplay(response, request, "viewer")
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}
