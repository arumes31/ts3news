package bot

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"ts3news/internal/config"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestLoginConsumesGrantAndSetsDistinctSessionCookies(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	mock.ExpectBegin()
	mock.ExpectQuery("DELETE FROM web_login_grants").
		WithArgs(authTokenDigest("one-time-grant")).
		WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("player-1"))
	mock.ExpectExec("INSERT INTO web_sessions").
		WithArgs(sqlmock.AnyArg(), "player-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	server := &WebServer{bot: &Bot{Cfg: &config.Config{WebBaseURL: "https://example.test"}, DB: database}}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/login?grant=one-time-grant&next=%2Fabyss", nil)
	server.handleLogin(response, request)
	cookies := response.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookies = %d, want 2", len(cookies))
	}
	auth := cookieByName(t, cookies, sessionCookie)
	expiry := cookieByName(t, cookies, sessionExpiryCookie)
	if !auth.HttpOnly || !auth.Secure || auth.Value == "" || auth.Value == "one-time-grant" {
		t.Fatalf("authentication cookie = %+v, want a distinct secure HttpOnly session", auth)
	}
	if auth.SameSite != http.SameSiteStrictMode || expiry.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookies must use SameSite=Strict: auth=%v expiry=%v", auth.SameSite, expiry.SameSite)
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
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if got := response.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q, want no-referrer", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestLoginGrantCannotBeReused(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	mock.ExpectBegin()
	mock.ExpectQuery("DELETE FROM web_login_grants").
		WithArgs(authTokenDigest("consumed-grant")).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()
	server := &WebServer{bot: &Bot{Cfg: &config.Config{WebBaseURL: "https://example.test"}, DB: database}}

	response := httptest.NewRecorder()
	server.handleLogin(response, httptest.NewRequest(http.MethodGet, "/login?grant=consumed-grant", nil))
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/denied" {
		t.Fatalf("response = %d %q, want redirect to denied", response.Code, response.Header().Get("Location"))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestSessionLookupUsesDigestAndExpiry(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	mock.ExpectQuery("SELECT client_uid FROM web_sessions").
		WithArgs(authTokenDigest("session-secret")).
		WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("player-1"))
	server := &WebServer{bot: &Bot{DB: database}}

	uid, ok := server.uidForSession("session-secret")
	if !ok || uid != "player-1" {
		t.Fatalf("uidForSession = %q, %v", uid, ok)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestLogoutRevokesSessionAndClearsCookies(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	mock.ExpectExec("DELETE FROM web_sessions").
		WithArgs(authTokenDigest("session-secret")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	server := &WebServer{bot: &Bot{Cfg: &config.Config{WebBaseURL: "https://example.test"}, DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/logout", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-secret"})
	response := httptest.NewRecorder()

	server.handleLogout(response, request)
	for _, name := range []string{sessionCookie, sessionExpiryCookie} {
		cookie := cookieByName(t, response.Result().Cookies(), name)
		if cookie.MaxAge != -1 || cookie.Value != "" {
			t.Errorf("cleared %s cookie = %+v", name, cookie)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(response, request)

	want := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "same-origin",
		"Permissions-Policy":     "camera=(), geolocation=(), microphone=()",
	}
	for name, value := range want {
		if got := response.Header().Get(name); got != value {
			t.Errorf("%s = %q, want %q", name, got, value)
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
