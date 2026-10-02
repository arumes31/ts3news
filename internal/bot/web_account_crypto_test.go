package bot

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAccountPasswordHash(t *testing.T) {
	password := "a long mobile passphrase"
	first, err := hashAccountPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	second, err := hashAccountPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || strings.Contains(first, password) {
		t.Fatal("password hashes must be independently salted")
	}
	if !verifyAccountPassword(first, password) || verifyAccountPassword(first, "wrong password") || verifyAccountPassword("", password) {
		t.Fatal("password verification failed")
	}
	for _, invalid := range []string{"short", strings.Repeat("x", 129)} {
		if _, err := hashAccountPassword(invalid); err == nil {
			t.Fatal("accepted invalid password")
		}
	}
}

func TestAccountCSRFBindsBrowserAndSession(t *testing.T) {
	s := &WebServer{}
	get := httptest.NewRequest(http.MethodGet, "/settings", nil)
	get.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-a"})
	response := httptest.NewRecorder()
	token := s.accountCSRF(response, get)
	form := url.Values{"csrf": {token}}
	request := func(session string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		for _, cookie := range response.Result().Cookies() {
			r.AddCookie(cookie)
		}
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: session})
		return r
	}
	if !s.validAccountForm(httptest.NewRecorder(), request("session-a")) {
		t.Fatal("valid form rejected")
	}
	if s.validAccountForm(httptest.NewRecorder(), request("session-b")) {
		t.Fatal("csrf crossed session boundary")
	}
	crossSite := request("session-a")
	crossSite.Header.Set("Sec-Fetch-Site", "cross-site")
	if s.validAccountForm(httptest.NewRecorder(), crossSite) {
		t.Fatal("cross-site form accepted")
	}
}

func TestAccountLoginDestination(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		{"local", "/abyss?tab=run", "/abyss?tab=run"},
		{"external", "https://evil.test", "/"},
		{"scheme relative", "//evil.test", "/"},
		{"backslash", "/\\evil.test", "/"},
		{"encoded", "/%5cevil.test", "/"},
		{"encoded authority", "/%2fevil.test", "/"},
		{"encoded carriage return", "/%0devil.test", "/"},
		{"opaque", "https:evil.test", "/"},
		{"empty", "", "/"},
		{"relative", "abyss", "/"},
		{"invalid escape", "/%zz", "/"},
		{"canonical local path", "/%61byss?tab=run#combat", "/abyss?tab=run#combat"},
		{"query with external URL", "/abyss?search=https%3A%2F%2Fexample.test", "/abyss?search=https%3A%2F%2Fexample.test"},
		{"empty query", "/abyss?", "/abyss?"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := accountDestination(tc.input); got != tc.want {
				t.Fatalf("got %q", got)
			}
		})
	}
}

func TestAccountLoginFormReferrerPolicy(t *testing.T) {
	s, err := NewWebServer(nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	s.handleAccountLogin(w, httptest.NewRequest(http.MethodGet, "/login", nil))
	if w.Code != http.StatusOK || w.Header().Get("Referrer-Policy") != "same-origin" {
		t.Fatal("login form must retain same-origin Origin headers for browser submissions")
	}
	if !strings.Contains(w.Body.String(), `autocomplete="current-password"`) {
		t.Fatal("password manager support missing")
	}
}

func TestAccountRecoveryExchangeSupportsExternalLinks(t *testing.T) {
	s := &WebServer{}
	w := httptest.NewRecorder()
	s.handleAccountRecovery(w, httptest.NewRequest(http.MethodGet, "/account/recover?token="+strings.Repeat("a", 64), nil))
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/account/password" || w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatal("recovery token was not privately exchanged")
	}
	cookie := cookieByName(t, w.Result().Cookies(), accountRecoveryCookie)
	if cookie.SameSite != http.SameSiteLaxMode || !cookie.HttpOnly || cookie.Path != "/account/password" || cookie.MaxAge != 600 {
		t.Fatal("recovery cookie must survive external top-level redirects and remain scoped to password setup")
	}
}
