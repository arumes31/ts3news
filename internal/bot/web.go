package bot

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"ts3news/internal/content"
	"ts3news/internal/games"
	"ts3news/internal/i18n"
	"ts3news/internal/leveling"
)

// jsonJS marshals v and returns it as template.JS so html/template injects it as
// a raw JS value inside <script> blocks. Passing a plain string instead makes
// html/template emit a *quoted* JS string, so the client sees a string where it
// expects an array/object.
func jsonJS(v any) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		return template.JS("null")
	}
	return template.JS(b)
}

//go:embed webassets/*.html webassets/*.css webassets/*.svg webassets/games/*.html
var webAssets embed.FS

const sessionCookie = "ts3session"

// WebServer is the player-facing portal: armoury, inventory, auto-battler,
// arcade and shop. It lives in the bot package so it can reuse the bot's stat,
// gear and loot helpers directly.
type WebServer struct {
	bot  *Bot
	tmpl *template.Template

	mu  sync.Mutex
	srv *http.Server
}

// NewWebServer parses the embedded templates and returns a ready server.
func NewWebServer(b *Bot) (*WebServer, error) {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"gold":  func(v int64) string { return FormatGoldPlain(v) },
		"comma": func(v int) string { return i18n.FormatLarge(float64(v)) },
		"lower": strings.ToLower,
		"seq": func(start, end int) []int {
			var out []int
			for i := start; i <= end; i++ {
				out = append(out, i)
			}
			return out
		},
		"mulpct": func(a, b int) int {
			if b <= 0 {
				return 0
			}
			p := a * 100 / b
			if p < 0 {
				return 0
			}
			if p > 100 {
				return 100
			}
			return p
		},
	}).ParseFS(webAssets, "webassets/*.html")
	if err != nil {
		return nil, fmt.Errorf("parsing web templates: %w", err)
	}
	return &WebServer{bot: b, tmpl: tmpl}, nil
}

// Start runs the HTTP server (blocking). Intended to be launched in a goroutine.
// When ctx is cancelled the server is gracefully shut down. Start returns nil on
// a clean shutdown so callers can distinguish it from a real listen error.
func (s *WebServer) Start(ctx context.Context, addr string) error {
	mux := http.NewServeMux()

	// Static assets.
	mux.HandleFunc("/static/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		b, _ := webAssets.ReadFile("webassets/style.css")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/static/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		b, _ := webAssets.ReadFile("webassets/favicon.svg")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/static/logo.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		b, _ := webAssets.ReadFile("webassets/logo.svg")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		b, _ := webAssets.ReadFile("webassets/favicon.svg")
		_, _ = w.Write(b)
	})

	// Public.
	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/logout", s.handleLogout)
	mux.HandleFunc("/denied", s.handleDenied)

	// Authenticated pages.
	mux.HandleFunc("/", s.auth(s.handleArmory))
	mux.HandleFunc("/inventory", s.auth(s.handleInventory))
	mux.HandleFunc("/battle", s.auth(s.handleBattlePage))
	mux.HandleFunc("/arcade", s.auth(s.handleArcadePage))
	mux.HandleFunc("/shop", s.auth(s.handleShopPage))
	mux.HandleFunc("/ah", s.auth(s.handleAHPage))
	mux.HandleFunc("/games", s.auth(s.handleArcade3DHub))
	mux.HandleFunc("/play/", s.auth(s.handleArcade3DPlay))

	// Authenticated JSON APIs.
	mux.HandleFunc("/api/tft/buy", s.auth(s.handleTFTBuy))
	mux.HandleFunc("/api/tft/reroll", s.authAPI(s.handleTFTReroll))
	mux.HandleFunc("/api/tft/place", s.authAPI(s.handleTFTPlace))
	mux.HandleFunc("/api/tft/sell", s.authAPI(s.handleTFTSell))
	mux.HandleFunc("/api/tft/equip", s.authAPI(s.handleTFTEquip))
	mux.HandleFunc("/api/tft/combat", s.authAPI(s.handleTFTCombat))
	mux.HandleFunc("/api/arcade/play", s.authAPI(s.handleArcadeAPI))
	mux.HandleFunc("/api/arcade/daily-spin", s.authAPI(s.handleDailySpinAPI))
	mux.HandleFunc("/api/arcade3d/reward", s.authAPI(s.handleArcade3DReward))
	mux.HandleFunc("/api/shop/exchange", s.auth(s.handleExchangeAPI))
	mux.HandleFunc("/api/shop/buy", s.auth(s.handleBuyAPI))
	mux.HandleFunc("/api/inventory/equip", s.auth(s.handleEquipAPI))
	mux.HandleFunc("/api/inventory/sell", s.auth(s.handleSellAPI))
	mux.HandleFunc("/api/ah/buy", s.auth(s.handleAHBuyAPI))
	mux.HandleFunc("/api/ah/list", s.auth(s.handleAHListAPI))

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	s.mu.Lock()
	s.srv = srv
	s.mu.Unlock()

	// Trigger a graceful shutdown when the parent context is cancelled.
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	log.Printf("Web portal listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully stops the HTTP server if it is running.
func (s *WebServer) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	srv := s.srv
	s.mu.Unlock()
	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
}

// ---- Auth ----------------------------------------------------------------

// ensureWebToken returns the user's persistent login token, generating and
// storing one on first use.
func (b *Bot) ensureWebToken(uid string) (string, error) {
	var tok *string
	err := b.DB.QueryRow("SELECT web_token FROM users WHERE client_uid=$1", uid).Scan(&tok)
	if err == nil && tok != nil && *tok != "" {
		return *tok, nil
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	newTok := hex.EncodeToString(raw)
	if _, err := b.DB.Exec("UPDATE users SET web_token=$1 WHERE client_uid=$2", newTok, uid); err != nil {
		return "", err
	}
	return newTok, nil
}

// loginURL builds the public login link for a token.
func (b *Bot) loginURL(token string) string {
	return fmt.Sprintf("%s/login?token=%s", b.Cfg.WebBaseURL, token)
}

// composeLoginPM builds the per-cycle private message containing the user's
// personal (shortened) web-portal login link. Returns "" on failure so the
// caller can simply skip it.
func (b *Bot) composeLoginPM(uid string) string {
	token, err := b.ensureWebToken(uid)
	if err != nil || token == "" {
		return ""
	}
	url := b.loginURL(token)
	if short, err := games.ShortenURL(url); err == nil && short != "" {
		url = short
	}
	return i18n.T("web.login_pm", url)
}

// uidForToken resolves a login token to a user UID.
func (s *WebServer) uidForToken(token string) (string, bool) {
	if token == "" {
		return "", false
	}
	var uid string
	err := s.bot.DB.QueryRow("SELECT client_uid FROM users WHERE web_token=$1", token).Scan(&uid)
	if err != nil {
		return "", false
	}
	return uid, true
}

// auth wraps a handler, resolving the session cookie to a user UID and passing
// it through. Unauthenticated requests are redirected to /denied.
func (s *WebServer) auth(h func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil {
			http.Redirect(w, r, "/denied", http.StatusSeeOther)
			return
		}
		uid, ok := s.uidForToken(c.Value)
		if !ok {
			http.Redirect(w, r, "/denied", http.StatusSeeOther)
			return
		}
		h(w, r, uid)
	}
}

// authAPI wraps a handler, resolving the session cookie to a user UID and passing
// it through. Unauthenticated requests get a JSON error.
func (s *WebServer) authAPI(h func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "unauthenticated"})
			return
		}
		uid, ok := s.uidForToken(c.Value)
		if !ok {
			writeJSON(w, map[string]any{"ok": false, "error": "unauthenticated"})
			return
		}
		h(w, r, uid)
	}
}

func (s *WebServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if _, ok := s.uidForToken(token); !ok {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}
	// Only flag the cookie Secure when the portal is served over HTTPS, so the
	// session token never leaks over plain HTTP in production deployments while
	// still working for local http:// development.
	secure := strings.HasPrefix(strings.ToLower(s.bot.Cfg.WebBaseURL), "https://")
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(90 * 24 * time.Hour),
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *WebServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/denied", http.StatusSeeOther)
}

func (s *WebServer) handleDenied(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	s.render(w, "denied", map[string]any{"Title": "Access"})
}

// ---- Shared model & rendering -------------------------------------------

type webUser struct {
	UID         string
	Nickname    string
	Level       int
	Prestige    int
	LevelName   string
	XP          int
	XPIntoLevel int
	XPForNext   int
	Gold        int64
	Scrap       int
	CurrentHP   int
	MaxHP       int
	Stats       content.Stats
	GearScore   int
}

// loadWebUser assembles the full character snapshot for a user.
func (s *WebServer) loadWebUser(uid string) (*webUser, error) {
	u := &webUser{UID: uid}
	var nick *string
	err := s.bot.DB.QueryRow(
		"SELECT nickname, level, prestige, xp, gold, scrap_stack, current_hp FROM users WHERE client_uid=$1", uid,
	).Scan(&nick, &u.Level, &u.Prestige, &u.XP, &u.Gold, &u.Scrap, &u.CurrentHP)
	if err != nil {
		return nil, err
	}
	if nick != nil {
		u.Nickname = *nick
	}
	if u.Level < 1 {
		u.Level = 1
	}

	stats, _, gearScore, _ := s.bot.calculateTotalStats(uid, time.Now())
	u.Stats = stats
	u.GearScore = gearScore
	u.MaxHP = stats.HP
	if u.CurrentHP <= 0 || u.CurrentHP > u.MaxHP {
		u.CurrentHP = u.MaxHP
	}
	u.LevelName = leveling.LevelName(u.Level)

	curBase := leveling.XPForLevel(u.Level)
	nextBase := leveling.XPForLevel(u.Level + 1)
	u.XPIntoLevel = u.XP - curBase
	u.XPForNext = nextBase - curBase
	if u.XPForNext < 1 {
		u.XPForNext = 1
	}
	return u, nil
}

// nav describes the active page for the shared navigation bar.
func (s *WebServer) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("web: render %s failed: %v", name, err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}
