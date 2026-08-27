package bot

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNormalizeAbyssFontSize(t *testing.T) {
	t.Parallel()
	for input, want := range map[string]string{"s": "s", " M ": "m", "L": "l", "huge": "m", "": "m"} {
		if got := normalizeAbyssFontSize(input); got != want {
			t.Errorf("normalizeAbyssFontSize(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestAbyssFontSizePreferenceRoundTrip(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	uid := "player"
	mock.ExpectQuery("SELECT value FROM app_meta").WithArgs(abyssFontSizeKey(uid)).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("l"))
	if got := (&Bot{DB: database}).loadAbyssFontSize(uid); got != "l" {
		t.Fatalf("font size = %q, want l", got)
	}
	mock.ExpectExec("INSERT INTO app_meta").WithArgs(abyssFontSizeKey(uid), "s").
		WillReturnResult(sqlmock.NewResult(1, 1))
	server := &WebServer{bot: &Bot{DB: database}}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/preferences/font-size", strings.NewReader(`{"font_size":"s"}`))
	server.handleAbyssFontSize(response, request, uid)
	if !strings.Contains(response.Body.String(), `"font_size":"s"`) {
		t.Fatalf("response = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssFontSizePreferenceRejectsInvalidValue(t *testing.T) {
	server := &WebServer{bot: &Bot{}}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/preferences/font-size", strings.NewReader(`{"font_size":"200%"}`))
	server.handleAbyssFontSize(response, request, "player")
	if !strings.Contains(response.Body.String(), `"ok":false`) {
		t.Fatalf("response = %s", response.Body.String())
	}
}
