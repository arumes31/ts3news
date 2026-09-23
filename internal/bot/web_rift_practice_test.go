package bot

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/rift"
)

func TestRiftPracticeScopeRejectsEconomicActionsBeforeStorage(t *testing.T) {
	b := &Bot{}
	for _, mode := range []string{"movement", "jump", "combo", "guard", "hazard", "boss"} {
		for _, kind := range []string{"bank", "next", "advance", "exit", "retry_boss"} {
			if _, err := b.updateRiftMode(context.Background(), "owner", riftRequest{Kind: kind}, rift.Build{}, time.Now(), mode); err == nil {
				t.Fatal("practice economic action accepted")
			}
		}
	}
	if _, err := b.updateRiftMode(context.Background(), "owner", riftRequest{Kind: "practice_reset"}, rift.Build{}, time.Now(), ""); err == nil {
		t.Fatal("practice reset entered campaign")
	}
	if _, err := b.updateRiftMode(context.Background(), "owner", riftRequest{Kind: "start"}, rift.Build{}, time.Now(), "unknown"); err == nil {
		t.Fatal("unknown practice accepted")
	}
}

func TestRiftPracticeStartWritesOnlyItsOwnAccountDrill(t *testing.T) {
	for _, mode := range []string{"movement", "jump", "combo", "guard", "hazard", "boss"} {
		t.Run(mode, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer database.Close()
			key := "rift_practice:owner:" + mode
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
			mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
			mock.ExpectQuery("SELECT value FROM app_meta").WithArgs(key).WillReturnError(sql.ErrNoRows)
			mock.ExpectExec("INSERT INTO app_meta").WithArgs(key, riftSnapshotCheck(func(run *rift.Run) bool {
				return run.Practice != nil && run.Practice.Mode == mode && run.Level == nil && len(run.History) == 0 && len(run.Drops) == 0
			})).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
			run, err := (&Bot{DB: database}).updateRiftMode(context.Background(), "owner", riftRequest{Kind: "start", RequestID: "practice-start-request"}, rift.Build{HP: 100}, time.Unix(100, 0), mode)
			if err != nil || run.Practice.Mode != mode {
				t.Fatalf("practice start failed: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRiftPracticeLoadRejectsCrossModeSnapshot(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	run, _ := rift.NewPracticeRun("practice", rift.Build{HP: 100}, "combo", time.Unix(100, 0))
	saved, _ := json.Marshal(run)
	mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("rift_practice:owner:movement").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(saved)))
	if _, err := loadRiftMode(context.Background(), database, "owner", "movement"); err == nil {
		t.Fatal("cross-mode snapshot accepted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRiftPracticeHTTPRejectsInvalidScopeBeforeBuild(t *testing.T) {
	for _, tc := range []struct{ method, url, body string }{
		{"GET", "/api/abyss/rift?practice=unknown", ""},
		{"POST", "/api/abyss/rift?practice=movement", `{"kind":"bank","request_id":"practice-request-123"}`},
		{"POST", "/api/abyss/rift", `{"kind":"practice_reset","request_id":"practice-request-123"}`},
	} {
		request := httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		(&WebServer{}).handleRiftAPI(response, request, "owner")
		if response.Code != 400 {
			t.Fatalf("%s: got %d", tc.url, response.Code)
		}
	}
}

func TestRiftPracticeResetIsScopedAndRevisionIdempotent(t *testing.T) {
	for _, replay := range []bool{false, true} {
		t.Run(map[bool]string{false: "reset", true: "replay"}[replay], func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer database.Close()
			run, _ := rift.NewPracticeRun("practice-run", rift.Build{HP: 100}, "combo", time.Unix(100, 0))
			run.Epoch = "2"
			run.Revision = 4
			run.Practice.Hits = 8
			run.Practice.Completed = true
			run.Status = "complete"
			saved, _ := json.Marshal(run)
			key := "rift_practice:owner:combo"
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
			mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
			mock.ExpectQuery("SELECT value FROM app_meta").WithArgs(key).WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(saved)))
			revision := 5
			if replay {
				revision = 4
				mock.ExpectRollback()
			} else {
				mock.ExpectExec("INSERT INTO app_meta").WithArgs(key, riftSnapshotCheck(func(saved *rift.Run) bool {
					return saved.ID == run.ID && saved.Revision == 5 && saved.Practice.Hits == 0 && !saved.Practice.Completed && saved.Status == "fighting" && len(saved.History) == 0
				})).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			}
			result, err := (&Bot{DB: database}).updateRiftMode(context.Background(), "owner", riftRequest{Kind: "practice_reset", RunID: run.ID, Revision: revision}, rift.Build{}, time.Unix(200, 0), "combo")
			if err != nil {
				t.Fatal(err)
			}
			if replay && result.Practice.Hits != 8 {
				t.Fatal("duplicate request reset progress")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRiftPracticeRecoveryCannotEnterCampaign(t *testing.T) {
	for _, kind := range []string{"practice_health", "practice_mana", "practice_cooldowns"} {
		if _, err := (&Bot{}).updateRift(context.Background(), "owner", riftRequest{Kind: kind}, rift.Build{}, time.Now()); err == nil {
			t.Fatal("campaign accepted practice tool")
		}
		request := httptest.NewRequest("POST", "/api/abyss/rift", strings.NewReader(`{"kind":"`+kind+`","request_id":"practice-request-123"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		(&WebServer{}).handleRiftAPI(response, request, "owner")
		if response.Code != 400 {
			t.Fatal("campaign HTTP accepted practice tool")
		}
	}
}
