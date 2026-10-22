package bot

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ts3news/internal/config"
)

func TestPortalGuardRejectsForeignStateChanges(t *testing.T) {
	t.Parallel()
	server := &WebServer{bot: &Bot{Cfg: &config.Config{WebBaseURL: "https://example.com"}}}
	for _, tc := range []struct {
		name    string
		method  string
		headers map[string]string
		want    bool
	}{
		{"same-origin fetch", http.MethodPost, map[string]string{"Sec-Fetch-Site": "same-origin", "Origin": "https://example.com"}, true},
		{"no browser headers", http.MethodPost, nil, true},
		{"cross-site fetch", http.MethodPost, map[string]string{"Sec-Fetch-Site": "cross-site"}, false},
		{"sibling subdomain", http.MethodPost, map[string]string{"Sec-Fetch-Site": "same-site"}, false},
		{"foreign origin", http.MethodPost, map[string]string{"Origin": "https://evil.example.net"}, false},
		{"plaintext origin on https portal", http.MethodPost, map[string]string{"Origin": "http://example.com"}, false},
		{"cross-site read", http.MethodGet, map[string]string{"Sec-Fetch-Site": "cross-site"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, "https://example.com/api/shop/buy", strings.NewReader("{}"))
			for key, value := range tc.headers {
				request.Header.Set(key, value)
			}
			if got := server.guardPortalRequest(httptest.NewRecorder(), request); got != tc.want {
				t.Fatalf("guardPortalRequest = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPortalGuardCapsRequestBodies(t *testing.T) {
	t.Parallel()
	server := &WebServer{bot: &Bot{Cfg: &config.Config{}}}
	request := httptest.NewRequest(http.MethodPost, "/api/shop/buy", strings.NewReader(strings.Repeat("a", portalBodyLimit+1)))
	if !server.guardPortalRequest(httptest.NewRecorder(), request) {
		t.Fatal("same-origin request rejected")
	}
	var tooLarge *http.MaxBytesError
	if _, err := io.ReadAll(request.Body); !errors.As(err, &tooLarge) {
		t.Fatalf("oversized body error = %v, want MaxBytesError", err)
	}
}

func TestReadJSONCapsRequestBodies(t *testing.T) {
	t.Parallel()
	body := `{"name":"` + strings.Repeat("a", portalBodyLimit) + `"}`
	var payload struct{ Name string }
	err := readJSON(httptest.NewRequest(http.MethodPost, "/api/x", strings.NewReader(body)), &payload)
	var tooLarge *http.MaxBytesError
	if !errors.As(err, &tooLarge) {
		t.Fatalf("readJSON error = %v, want MaxBytesError", err)
	}
}

func TestLogoutRejectsGet(t *testing.T) {
	t.Parallel()
	server := &WebServer{bot: &Bot{Cfg: &config.Config{}}}
	response := httptest.NewRecorder()
	server.handleLogout(response, httptest.NewRequest(http.MethodGet, "/logout", nil))
	if response.Code != http.StatusMethodNotAllowed || len(response.Result().Cookies()) != 0 {
		t.Fatalf("GET logout status=%d cookies=%d, want 405 and no cookie changes", response.Code, len(response.Result().Cookies()))
	}
}
