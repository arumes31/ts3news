//go:build integration

package bot

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"ts3news/internal/config"
)

func accountTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("ACCOUNT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set ACCOUNT_TEST_DATABASE_URL to a disposable PostgreSQL database")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("account_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := sql.Open("postgres", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`CREATE TABLE users(client_uid TEXT PRIMARY KEY, nickname TEXT, web_token TEXT, web_token_expires TIMESTAMPTZ);
 INSERT INTO users VALUES ('player-one','Player One','legacy-one',NOW()+INTERVAL '1 day'), ('player-two','Player Two','legacy-two',NULL);`)
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../db/migrations/0105_web_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAccountMigrationIntegration(t *testing.T) {
	db := accountTestDatabase(t)
	_, err := db.Exec(`INSERT INTO users(client_uid) SELECT 'generated-' || n FROM generate_series(1,2500) n`)
	if err != nil {
		t.Fatal(err)
	}
	var count, unique, readable int
	err = db.QueryRow(`SELECT COUNT(*), COUNT(DISTINCT web_username), COUNT(*) FILTER (WHERE web_username ~ '^[a-z]+-[a-z]+-[0-9]+$') FROM users`).Scan(&count, &unique, &readable)
	if err != nil || count != 2502 || unique != count || readable != count {
		t.Fatalf("username assignment: %d/%d/%d %v", count, unique, readable, err)
	}
	var before, after string
	db.QueryRow("SELECT web_username FROM users WHERE client_uid='player-one'").Scan(&before)
	if _, err := db.Exec("UPDATE users SET nickname='New nickname' WHERE client_uid='player-one'"); err != nil {
		t.Fatal(err)
	}
	db.QueryRow("SELECT web_username FROM users WHERE client_uid='player-one'").Scan(&after)
	if before != after {
		t.Fatal("nickname changed login identifier")
	}
	if _, err := db.Exec("INSERT INTO users (client_uid, web_username) VALUES ('collision',$1)", strings.ToUpper(before)); err == nil {
		t.Fatal("case-insensitive collision accepted")
	}
	var migrated int
	if err := db.QueryRow("SELECT COUNT(*) FROM web_sessions WHERE expires_at > NOW()").Scan(&migrated); err != nil || migrated != 2 {
		t.Fatalf("legacy migration: %d %v", migrated, err)
	}
}

type accountTestBrowser struct {
	t      *testing.T
	client *http.Client
	base   string
}

func newAccountTestBrowser(t *testing.T, base string) *accountTestBrowser {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &accountTestBrowser{t: t, base: base, client: &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (b *accountTestBrowser) request(method, path string, form url.Values) (int, string, string) {
	b.t.Helper()
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	r, err := http.NewRequest(method, b.base+path, body)
	if err != nil {
		b.t.Fatal(err)
	}
	if form != nil {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", b.base)
	}
	response, err := b.client.Do(r)
	if err != nil {
		b.t.Fatal(err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		b.t.Fatal(err)
	}
	return response.StatusCode, string(raw), response.Header.Get("Location")
}
func (b *accountTestBrowser) form(path string) url.Values {
	b.t.Helper()
	code, body, _ := b.request("GET", path, nil)
	if code != 200 {
		b.t.Fatalf("GET %s: %d %s", path, code, body)
	}
	match := regexp.MustCompile(`name="csrf" value="([^"]+)"`).FindStringSubmatch(body)
	if len(match) != 2 {
		b.t.Fatalf("no form on %s: %.300s", path, body)
	}
	return url.Values{"csrf": {match[1]}}
}
func (b *accountTestBrowser) session() string {
	u, _ := url.Parse(b.base)
	for _, c := range b.client.Jar.Cookies(u) {
		if c.Name == sessionCookie {
			return c.Value
		}
	}
	return ""
}

func TestAccountDeviceAndRecoveryIntegration(t *testing.T) {
	db := accountTestDatabase(t)
	bot := &Bot{DB: db, Cfg: &config.Config{}}
	server, err := NewWebServer(bot)
	if err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer(server.routes())
	defer host.Close()
	bot.Cfg.WebBaseURL = host.URL
	a, b := newAccountTestBrowser(t, host.URL), newAccountTestBrowser(t, host.URL)
	for _, browser := range []*accountTestBrowser{a, b} {
		code, body, next := browser.request("GET", "/login?token=legacy-one&next=%2Fsettings", nil)
		if code != 303 || next != "/settings" {
			t.Fatalf("TeamSpeak login %d %.200s", code, body)
		}
	}
	if a.session() == b.session() || a.session() == "legacy-one" {
		t.Fatal("devices share credentials")
	}
	_, settings, _ := a.request("GET", "/settings", nil)
	if !strings.Contains(settings, "!password") || !strings.Contains(settings, "Password not set") {
		t.Fatal("enrollment instructions missing")
	}
	token, err := bot.issueAccountRecovery(context.Background(), "player-one")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bot.issueAccountRecovery(context.Background(), "player-one"); err == nil {
		t.Fatal("recovery request cooldown bypassed")
	}
	reset := newAccountTestBrowser(t, host.URL)
	code, _, next := reset.request("GET", "/account/recover?token="+token, nil)
	if code != 303 || next != "/account/password" {
		t.Fatal("recovery exchange failed")
	}
	form := reset.form(next)
	form.Set("password", "my first mobile passphrase")
	form.Set("confirm_password", "does not match")
	code, body, _ := reset.request("POST", next, form)
	if code != 200 || !strings.Contains(body, "do not match") {
		t.Fatal("mismatching password accepted")
	}
	form = reset.form(next)
	form.Set("password", "my first mobile passphrase")
	form.Set("confirm_password", form.Get("password"))
	if code, body, _ := reset.request("POST", next, form); code != 303 {
		t.Fatalf("enrollment %d %s", code, body)
	}
	for _, browser := range []*accountTestBrowser{a, b} {
		if code, _, _ := browser.request("GET", "/settings", nil); code != 303 {
			t.Fatal("old session survived reset")
		}
	}
	if code, body, _ := a.request("GET", "/login?token=legacy-one", nil); code != 200 || !strings.Contains(body, "expired") {
		t.Fatal("old TeamSpeak link survived reset")
	}
	reset.request("GET", "/account/recover?token="+token, nil)
	if _, body, _ := reset.request("GET", "/account/password", nil); !strings.Contains(body, "already used") {
		t.Fatal("recovery link reused")
	}
	var username string
	if err := db.QueryRow("SELECT web_username FROM users WHERE client_uid='player-one'").Scan(&username); err != nil {
		t.Fatal(err)
	}
	login := func(browser *accountTestBrowser, password string) int {
		t.Helper()
		f := browser.form("/login")
		f.Set("username", " "+strings.ToUpper(username)+" ")
		f.Set("password", password)
		f.Set("next", "/settings")
		code, _, _ := browser.request("POST", "/login", f)
		return code
	}
	if login(a, "wrong password") != 200 {
		t.Fatal("wrong-password response")
	}
	if login(a, "my first mobile passphrase") != 303 || login(b, "my first mobile passphrase") != 303 {
		t.Fatal("mobile login failed")
	}
	a.request("GET", "/logout", nil)
	if code, _, _ := b.request("GET", "/settings", nil); code != 200 {
		t.Fatal("logout revoked another device")
	}
	if code, _, _ := a.request("GET", "/settings", nil); code != 303 {
		t.Fatal("logout did not revoke current device")
	}
	f := b.form("/settings")
	f.Set("current_password", "my first mobile passphrase")
	f.Set("password", "my replacement mobile passphrase")
	f.Set("confirm_password", f.Get("password"))
	if code, body, _ := b.request("POST", "/settings", f); code != 303 {
		t.Fatalf("password change %d %s", code, body)
	}
	if login(a, "my first mobile passphrase") != 200 || login(a, "my replacement mobile passphrase") != 303 {
		t.Fatal("password change did not replace credentials")
	}
	var uid string
	if err := db.QueryRow("SELECT client_uid FROM web_sessions WHERE token_hash=$1", accountDigest(a.session())).Scan(&uid); err != nil || uid != "player-one" {
		t.Fatal("login did not preserve original identity")
	}
}

func TestAccountRecoveryConcurrentConsumptionIntegration(t *testing.T) {
	db := accountTestDatabase(t)
	bot := &Bot{DB: db, Cfg: &config.Config{}}
	s, err := NewWebServer(bot)
	if err != nil {
		t.Fatal(err)
	}
	token, err := bot.issueAccountRecovery(context.Background(), "player-one")
	if err != nil {
		t.Fatal(err)
	}
	hash, err := hashAccountPassword("concurrent reset passphrase")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() {
			results <- s.saveAccountPassword(httptest.NewRecorder(), httptest.NewRequest("POST", "/account/password", nil), "", "", hash, token)
		})
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if err != sql.ErrNoRows {
			t.Fatal(err)
		}
	}
	if success != 1 {
		t.Fatalf("recovery consumed %d times", success)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM web_sessions WHERE client_uid='player-one'").Scan(&count); err != nil || count != 1 {
		t.Fatal("reset left incorrect sessions")
	}
}

func TestAccountSessionBoundAndLegacyLogoutIntegration(t *testing.T) {
	db := accountTestDatabase(t)
	bot := &Bot{DB: db, Cfg: &config.Config{}}
	s, err := NewWebServer(bot)
	if err != nil {
		t.Fatal(err)
	}
	for range 25 {
		if err := s.finishAccountLogin(httptest.NewRecorder(), httptest.NewRequest("GET", "/login", nil), "", "legacy-one", false); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM web_sessions WHERE client_uid='player-one'").Scan(&count); err != nil || count != 20 {
		t.Fatalf("session bound %d %v", count, err)
	}
	if err := s.revokeAccountSession(context.Background(), "legacy-one"); err != nil {
		t.Fatal(err)
	}
	if err := s.finishAccountLogin(httptest.NewRecorder(), httptest.NewRequest("GET", "/login", nil), "", "legacy-one", false); err != sql.ErrNoRows {
		t.Fatalf("legacy logout credential replay: %v", err)
	}
}

func TestAccountRecoverySurvivesLoginThrottleIntegration(t *testing.T) {
	db := accountTestDatabase(t)
	bot := &Bot{DB: db, Cfg: &config.Config{}}
	s, err := NewWebServer(bot)
	if err != nil {
		t.Fatal(err)
	}
	var username string
	if err := db.QueryRow("SELECT web_username FROM users WHERE client_uid='player-one'").Scan(&username); err != nil {
		t.Fatal(err)
	}
	for range 11 {
		if _, err := s.accountAttempt(context.Background(), "account:login:"+username, 10); err != nil {
			t.Fatal(err)
		}
	}
	token, err := bot.issueAccountRecovery(context.Background(), "player-one")
	if err != nil {
		t.Fatal(err)
	}
	if !s.accountThrottle(httptest.NewRecorder(), httptest.NewRequest("POST", "/account/password", nil), "recovery:"+accountDigest(token)) {
		t.Fatal("anonymous login failures blocked verified recovery")
	}
	if _, err := db.Exec("UPDATE users SET web_recovery_expires=NOW()-INTERVAL '1 second' WHERE client_uid='player-one'"); err != nil {
		t.Fatal(err)
	}
	hash, err := hashAccountPassword("expired recovery passphrase")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.saveAccountPassword(httptest.NewRecorder(), httptest.NewRequest("POST", "/account/password", nil), "", "", hash, token); err != sql.ErrNoRows {
		t.Fatal("expired recovery was accepted")
	}
}

// Opt-in localhost fixture using the production router and a disposable schema.
func TestAccountBrowserServer(t *testing.T) {
	if os.Getenv("ACCOUNT_BROWSER_SERVER") != "1" {
		t.Skip("browser fixture is opt-in")
	}
	db := accountTestDatabase(t)
	hash, err := hashAccountPassword("playtest mobile passphrase")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE users SET web_password_hash=$1 WHERE client_uid='player-one'", hash); err != nil {
		t.Fatal(err)
	}
	// Synthetic recovery credential for browser checks, valid only in this fixture.
	if _, err := db.Exec("UPDATE users SET web_recovery_hash=$1, web_recovery_expires=NOW()+INTERVAL '10 minutes' WHERE client_uid='player-two'", accountDigest(strings.Repeat("a", 64))); err != nil {
		t.Fatal(err)
	}
	s, err := NewWebServer(&Bot{DB: db, Cfg: &config.Config{WebBaseURL: "http://127.0.0.1:55440"}})
	if err != nil {
		t.Fatal(err)
	}
	mux := s.routes()
	mux.HandleFunc("GET /__test/link", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<a href="http://127.0.0.1:55440/account/recover?token=%s">Open test TeamSpeak recovery link</a>`, strings.Repeat("a", 64))
	})
	stop := make(chan struct{})
	var once sync.Once
	mux.HandleFunc("POST /__test/shutdown", func(w http.ResponseWriter, r *http.Request) { once.Do(func() { close(stop) }); w.WriteHeader(204) })
	listener, err := net.Listen("tcp", "127.0.0.1:55440")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go server.Serve(listener)
	defer server.Close()
	t.Log("account browser fixture ready on localhost:55440")
	select {
	case <-stop:
	case <-time.After(20 * time.Minute):
	}
}
