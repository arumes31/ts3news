package bot

import (
	"context"
	"encoding/json"
	"github.com/DATA-DOG/go-sqlmock"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"ts3news/internal/rift"
)

func TestRiftPracticeEnemyActionsAreScopedAndReplaySafe(t *testing.T) {
	for _, kind := range []string{"practice_spawn", "practice_clear"} {
		for _, replay := range []bool{false, true} {
			t.Run(kind+map[bool]string{false: "/new", true: "/replay"}[replay], func(t *testing.T) {
				request := riftRequest{Kind: kind, RequestID: "practice-request-123", EnemyName: riftMobCatalog(time.Now())[0].Name}
				if !validRiftRequest(request) || !validRiftModeAction("skills", kind) || validRiftModeAction("", kind) || validRiftModeAction("combo", kind) {
					t.Fatal("invalid action scope")
				}
				database, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = database.Close() }()
				run, _ := rift.NewPracticeRun("practice", rift.Build{HP: 100}, "skills", time.Unix(100, 0))
				run.Epoch = "2"
				run.Revision = 4
				if replay { run.LastRequestID = request.RequestID }
				saved, _ := json.Marshal(run)
				key := "rift_practice:owner:skills"
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
				mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
				mock.ExpectQuery("SELECT value FROM app_meta").WithArgs(key).WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(saved)))
				request.RunID = run.ID
				request.Revision = 5
				if replay {
					request.Revision = 4
					mock.ExpectRollback()
				} else {
					mock.ExpectExec("INSERT INTO app_meta").WithArgs(key, riftSnapshotCheck(func(result *rift.Run) bool {
						return result.Revision == 5 && result.Practice.Mode == "skills" && result.Gold == 0 && len(result.Drops) == 0 && ((kind == "practice_clear" && len(result.Enemies) == 0) || (kind == "practice_spawn" && len(result.Enemies) == 1 && result.Enemies[0].Name == request.EnemyName))
					})).WillReturnResult(sqlmock.NewResult(0, 1))
					mock.ExpectCommit()
				}
				result, err := (&Bot{DB: database}).updateRiftMode(context.Background(), "owner", request, rift.Build{}, time.Unix(200, 0), "skills")
				if err != nil {
					t.Fatal(err)
				}
				if replay {
					after, _ := json.Marshal(result)
					if string(after) != string(saved) {
						t.Fatal("replay changed saved state")
					}
				}
				if err = mock.ExpectationsWereMet(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
	for _, name := range []string{"", strings.Repeat("x", 513)} {
		if validRiftRequest(riftRequest{Kind: "practice_spawn", EnemyName: name, RequestID: "practice-request-123"}) {
			t.Fatal("invalid enemy name accepted")
		}
	}
}

func TestRiftPracticeEnemyHTTPRejectsOtherModesBeforeBuild(t *testing.T) {
	for _, mode := range []string{"", "combo", "boss", "resource"} {
		for _, kind := range []string{"practice_spawn", "practice_clear"} {
			raw, _ := json.Marshal(riftRequest{Kind: kind, EnemyName: "Snotty Rat", RequestID: "practice-request-123"})
			request := httptest.NewRequest("POST", "/api/abyss/rift?practice="+mode, strings.NewReader(string(raw)))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			(&WebServer{}).handleRiftAPI(response, request, "owner")
			if response.Code != 400 {
				t.Fatalf("mode %q action %q: status %d", mode, kind, response.Code)
			}
		}
	}
}
