package bot

import (
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewAbyssAPITokenIsOpaqueAndSelfConsistent(t *testing.T) {
	t.Parallel()

	token, hash, prefix, err := newAbyssAPIToken()
	if err != nil {
		t.Fatalf("newAbyssAPIToken: %v", err)
	}
	if !strings.HasPrefix(token, abyssAPITokenPrefix) || prefix != token[:12] {
		t.Fatalf("token=%q prefix=%q", token, prefix)
	}
	digest := sha256.Sum256([]byte(token))
	if string(hash) != string(digest[:]) {
		t.Fatal("stored token digest does not match the plaintext token")
	}
}

func TestAuthenticateAbyssAPITokenUsesStoredDigest(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	token := "abp_abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
	digest := sha256.Sum256([]byte(token))
	mock.ExpectQuery("SELECT token_hash FROM abyss_api_tokens").WithArgs(token[:12]).
		WillReturnRows(sqlmock.NewRows([]string{"token_hash"}).AddRow(digest[:]))

	server := &WebServer{bot: &Bot{DB: database}}
	if err := server.authenticateAbyssAPIToken(t.Context(), "Bearer "+token); err != nil {
		t.Fatalf("authenticateAbyssAPIToken: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAbyssTokenStatsRequiresBearerToken(t *testing.T) {
	server := &WebServer{bot: &Bot{}}
	request := httptest.NewRequest(http.MethodGet, "/api/abyss/stats", nil)
	response := httptest.NewRecorder()
	server.handleAbyssTokenStats(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestHandleAbyssAPITokenPostRequiresJSON(t *testing.T) {
	server := &WebServer{bot: &Bot{}}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/abyss/api-token",
		strings.NewReader("rotate=true"),
	)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	server.handleAbyssAPIToken(response, request, "user-1")

	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}
