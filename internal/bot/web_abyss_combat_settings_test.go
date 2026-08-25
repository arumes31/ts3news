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
	get := httptest.NewRecorder()
	server.handleAbyssCombatSettings(get, httptest.NewRequest(http.MethodGet, "/api/abyss/combat/settings", nil), "user")
	if !strings.Contains(get.Body.String(), `"hold_mana":true`) {
		t.Fatalf("GET settings = %s", get.Body.String())
	}

	mock.ExpectExec(`INSERT INTO app_meta`).
		WithArgs("abyss_hold_mana:user", "1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	post := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/combat/settings", bytes.NewBufferString(`{"hold_mana":true}`))
	server.handleAbyssCombatSettings(post, request, "user")
	if !strings.Contains(post.Body.String(), `"hold_mana":true`) {
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
	mock.ExpectExec(`DELETE FROM app_meta`).
		WithArgs("abyss_hold_mana:user").
		WillReturnError(http.ErrServerClosed)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/combat/settings", bytes.NewBufferString(`{"hold_mana":false}`))
	(&WebServer{bot: &Bot{DB: db}}).handleAbyssCombatSettings(recorder, request, "user")
	if !strings.Contains(recorder.Body.String(), `"ok":false`) {
		t.Fatalf("failure response = %s", recorder.Body.String())
	}
}
