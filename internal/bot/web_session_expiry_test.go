package bot

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"ts3news/internal/config"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestLoginSetsPrivateSessionAndPublicExpiryCookies(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	mock.ExpectQuery("SELECT client_uid FROM users WHERE web_token=\\$1").
		WithArgs("secret-token").
		WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("player-1"))
	mock.ExpectExec("UPDATE users SET web_token_expires=\\$1 WHERE web_token=\\$2").
		WithArgs(sqlmock.AnyArg(), "secret-token").
		WillReturnResult(sqlmock.NewResult(0, 1))
	server := &WebServer{bot: &Bot{Cfg: &config.Config{WebBaseURL: "https://example.test"}, DB: database}}

	response := httptest.NewRecorder()
	server.handleLogin(response, httptest.NewRequest(http.MethodGet, "/login?token=secret-token&next=%2Fabyss", nil))
	cookies := response.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookies = %d, want 2", len(cookies))
	}
	auth := cookieByName(t, cookies, sessionCookie)
	expiry := cookieByName(t, cookies, sessionExpiryCookie)
	if !auth.HttpOnly || !auth.Secure || auth.Value != "secret-token" {
		t.Fatalf("authentication cookie = %+v, want secure HttpOnly token", auth)
	}
	if expiry.HttpOnly || !expiry.Secure {
		t.Fatalf("expiry cookie = %+v, want secure JavaScript-readable timestamp", expiry)
	}
	expiryUnix, err := strconv.ParseInt(expiry.Value, 10, 64)
	if err != nil {
		t.Fatalf("parse expiry cookie: %v", err)
	}
	if delta := time.Unix(expiryUnix, 0).Sub(auth.Expires); delta < -time.Second || delta > time.Second {
		t.Fatalf("expiry companion differs from authentication expiry by %s", delta)
	}
	if location := response.Header().Get("Location"); location != "/abyss" {
		t.Fatalf("redirect = %q, want /abyss", location)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestLogoutClearsSessionAndExpiryCookies(t *testing.T) {
	t.Parallel()

	server := &WebServer{bot: &Bot{Cfg: &config.Config{WebBaseURL: "https://example.test"}}}
	response := httptest.NewRecorder()
	server.handleLogout(response, httptest.NewRequest(http.MethodPost, "/logout", nil))
	cookies := response.Result().Cookies()
	for _, name := range []string{sessionCookie, sessionExpiryCookie} {
		cookie := cookieByName(t, cookies, name)
		if cookie.MaxAge != -1 || cookie.Value != "" {
			t.Errorf("cleared %s cookie = %+v", name, cookie)
		}
	}
}

func cookieByName(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %q not found", name)
	return nil
}
