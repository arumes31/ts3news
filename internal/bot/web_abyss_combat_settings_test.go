package bot

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssCombatSettingsRoundTrip(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	server := &WebServer{bot: &Bot{DB: db}}

	mock.ExpectQuery(`SELECT value FROM app_meta WHERE key=\$1`).
		WithArgs("abyss_hold_mana:user").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("1"))
	mock.ExpectQuery(`SELECT value FROM app_meta WHERE key=\$1`).
		WithArgs("abyss_pet_command:user").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("guard"))
	get := httptest.NewRecorder()
	server.handleAbyssCombatSettings(get, httptest.NewRequest(http.MethodGet, "/api/abyss/combat/settings", nil), "user")
	if !strings.Contains(get.Body.String(), `"hold_mana":true`) || !strings.Contains(get.Body.String(), `"pet_command":"guard"`) {
		t.Fatalf("GET settings = %s", get.Body.String())
	}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO app_meta`).
		WithArgs("abyss_hold_mana:user", "1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO app_meta`).
		WithArgs("abyss_pet_command:user", "focus").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	post := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/combat/settings", bytes.NewBufferString(`{"hold_mana":true,"pet_command":"focus"}`))
	server.handleAbyssCombatSettings(post, request, "user")
	if !strings.Contains(post.Body.String(), `"hold_mana":true`) || !strings.Contains(post.Body.String(), `"pet_command":"focus"`) {
		t.Fatalf("POST settings = %s", post.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssCombatSettingsRejectsPersistenceFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM app_meta`).
		WithArgs("abyss_hold_mana:user").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO app_meta`).
		WithArgs("abyss_pet_command:user", "guard").
		WillReturnError(http.ErrServerClosed)
	mock.ExpectRollback()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/combat/settings", bytes.NewBufferString(`{"hold_mana":false,"pet_command":"guard"}`))
	(&WebServer{bot: &Bot{DB: db}}).handleAbyssCombatSettings(recorder, request, "user")
	if !strings.Contains(recorder.Body.String(), `"ok":false`) {
		t.Fatalf("failure response = %s", recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssCombatSettingsRejectsUnknownPetCommand(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/combat/settings", bytes.NewBufferString(`{"pet_command":"forged"}`))
	(&WebServer{bot: &Bot{}}).handleAbyssCombatSettings(recorder, request, "user")
	if !strings.Contains(recorder.Body.String(), `"error":"invalid companion command"`) {
		t.Fatalf("invalid command response = %s", recorder.Body.String())
	}
}

func TestAbyssCombatSettingsRetainsCommandForRollingClient(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	mock.ExpectQuery(`SELECT value FROM app_meta WHERE key=\$1`).
		WithArgs("abyss_pet_command:user").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("guard"))
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM app_meta`).
		WithArgs("abyss_hold_mana:user").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO app_meta`).
		WithArgs("abyss_pet_command:user", "guard").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/combat/settings", bytes.NewBufferString(`{"hold_mana":false}`))
	(&WebServer{bot: &Bot{DB: db}}).handleAbyssCombatSettings(recorder, request, "user")
	if !strings.Contains(recorder.Body.String(), `"pet_command":"guard"`) {
		t.Fatalf("rolling client response = %s", recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
