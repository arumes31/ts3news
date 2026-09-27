package bot

import (
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRiftDeletedCharacterStopsMutations(t *testing.T) {
	for _, kind := range []string{"step", "pause", "resume", "bank", "next", "advance", "exit"} {
		t.Run(kind, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer database.Close()
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("deleted-owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}))
			mock.ExpectRollback()
			request := httptest.NewRequest(http.MethodPost, "https://game.test/api/abyss/rift", strings.NewReader(`{"kind":"`+kind+`","run_id":"existing-run","request_id":"missing-character-request","revision":2}`))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			(&WebServer{bot: &Bot{DB: database}}).handleRiftAPI(response, request, "deleted-owner")
			if response.Code != http.StatusGone || !strings.Contains(response.Body.String(), `"code":"CHARACTER_MISSING"`) {
				t.Fatalf("%d: %s", response.Code, response.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRiftCharacterDatabaseFailureIsNotDeletion(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT client_uid FROM users").WillReturnError(errors.New("private database failure"))
	mock.ExpectRollback()
	request := httptest.NewRequest(http.MethodPost, "https://game.test/api/abyss/rift", strings.NewReader(`{"kind":"step","run_id":"existing-run","request_id":"database-error-request","revision":2}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	(&WebServer{bot: &Bot{DB: database}}).handleRiftAPI(response, request, "owner")
	if response.Code != 500 || strings.Contains(response.Body.String(), "CHARACTER_MISSING") || strings.Contains(response.Body.String(), "private database") {
		t.Fatalf("%d: %s", response.Code, response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRiftDeletedCharacterLoadAndStart(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer database.Close()
			empty := func(query string, columns ...string) {
				mock.ExpectQuery(query).WillReturnRows(sqlmock.NewRows(columns))
			}
			mock.ExpectQuery("SELECT level, prestige").WillReturnRows(sqlmock.NewRows([]string{"level", "prestige"}).AddRow(20, 0))
			mock.ExpectQuery("SELECT title, title_mult, title_expires").WillReturnRows(sqlmock.NewRows([]string{"title", "title_mult", "title_expires"}).AddRow(nil, nil, nil))
			mock.ExpectQuery("SELECT artifact_mult, artifact_name, artifact_durability").WillReturnRows(sqlmock.NewRows([]string{"artifact_mult", "artifact_name", "artifact_durability"}).AddRow(nil, nil, 0))
			empty("SELECT slot, gear_id, durability, enchantment_id, item_data", "slot", "gear_id", "durability", "enchantment_id", "item_data")
			mock.ExpectQuery("SELECT abyss_win_streak").WillReturnRows(sqlmock.NewRows([]string{"abyss_win_streak"}).AddRow(0))
			empty("SELECT skill_id FROM user_skills", "skill_id")
			empty("SELECT ultimate_id, current_cooldown", "ultimate_id", "current_cooldown")
			empty("SELECT cons_id, remaining_fights", "cons_id", "remaining_fights")
			empty("SELECT node_id FROM user_abyss_tree", "node_id")
			empty("SELECT value FROM app_meta", "value")

			mock.ExpectQuery("SELECT nickname, level").WithArgs("deleted-owner").WillReturnRows(sqlmock.NewRows([]string{"nickname", "level"}))
			request := httptest.NewRequest(method, "https://game.test/api/abyss/rift", strings.NewReader(`{"kind":"start","request_id":"missing-character-start","level_id":1}`))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			(&WebServer{bot: &Bot{DB: database}}).handleRiftAPI(response, request, "deleted-owner")
			if response.Code != http.StatusGone || !strings.Contains(response.Body.String(), `"code":"CHARACTER_MISSING"`) {
				t.Fatalf("%d: %s", response.Code, response.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
