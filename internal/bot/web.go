package bot

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
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
	return template.JS(b) // #nosec G203 - trusted JSON data from server, not user input
}

//go:embed webassets/*.html webassets/*.css webassets/*.svg webassets/icons/*.svg
var webAssets embed.FS

// styleCSSVer is a content hash of the embedded stylesheet, appended as a
// ?v= query to its <link> URL. Cloudflare and browsers cache /static/style.css
// aggressively, so without this a deploy can serve new HTML against a stale
// stylesheet and break page layouts.
var styleCSSVer = func() string {
	b, _ := webAssets.ReadFile("webassets/style.css")
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:6])
}()

const sessionCookie = "ts3session"

// WebServer is the player-facing portal: armoury, inventory, auto-battler,
// arcade and shop. It lives in the bot package so it can reuse the bot's stat,
// gear and loot helpers directly.
type WebServer struct {
	bot  *Bot
	tmpl *template.Template

	mu  sync.Mutex
	srv *http.Server

	// abyssLocks serialises each player's Abyss actions (enter/descend/revive/
	// concede/bank). The combat engine writes through b.DB directly and can't be
	// wrapped in one SQL transaction with the surrounding bookkeeping, so a
	// per-uid lock is what prevents the double-bank and post-death-descend races.
	abyssLocks sync.Map // uid -> *sync.Mutex
}

// lockAbyss acquires the per-uid Abyss mutex and returns its unlock func.
func (s *WebServer) lockAbyss(uid string) func() {
	v, _ := s.abyssLocks.LoadOrStore(uid, &sync.Mutex{})
	m := v.(*sync.Mutex)
	m.Lock()
	return m.Unlock
}

// NewWebServer parses the embedded templates and returns a ready server.
func NewWebServer(b *Bot) (*WebServer, error) {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"gold":  func(v int64) string { return FormatGoldPlain(v) },
		"comma": func(v any) string {
			var f float64
			switch n := v.(type) {
			case int:
				f = float64(n)
			case int32:
				f = float64(n)
			case int64:
				f = float64(n)
			case uint:
				f = float64(n)
			case uint32:
				f = float64(n)
			case uint64:
				f = float64(n)
			case float64:
				f = n
			case float32:
				f = float64(n)
			default:
				return fmt.Sprintf("%v", v)
			}
			return i18n.FormatLarge(f)
		},
		"T":     func(key string, args ...any) string { return i18n.T(key, args...) },
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
		"jsonJS": jsonJS,
		"mulf":   func(a, b float64) float64 { return a * b },
		"cssver": func() string { return styleCSSVer },
		"halve":  func(n int) int { return n / 2 },
		// dict builds a map from alternating key/value pairs, for passing several
		// named values into a sub-template (used by the Abyss upgrade widget).
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict: odd number of arguments")
			}
			m := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				k, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict: key %d is not a string", i)
				}
				m[k] = values[i+1]
			}
			return m, nil
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
	mux.HandleFunc("/static/style.css", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		// Long cache is safe: the <link> URL carries a content-hash ?v= that
		// changes on every stylesheet edit, so deploys bust the cache themselves.
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		b, _ := webAssets.ReadFile("webassets/style.css")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/static/favicon.svg", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		b, _ := webAssets.ReadFile("webassets/favicon.svg")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/static/logo.svg", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		b, _ := webAssets.ReadFile("webassets/logo.svg")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		b, _ := webAssets.ReadFile("webassets/favicon.svg")
		_, _ = w.Write(b)
	})
	// game-icons.net SVGs (CC BY 3.0), themed via CSS mask.
	mux.HandleFunc("/static/icons/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/static/icons/")
		if strings.Contains(name, "/") || !strings.HasSuffix(name, ".svg") {
			http.NotFound(w, r)
			return
		}
		b, err := webAssets.ReadFile("webassets/icons/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(b) // #nosec G705 - static SVG icon file, no user input
	})
	// Game common assets (animation-framework.js, game-framework-enhanced.js)
	mux.HandleFunc("/play/common/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/play/common/")
		if strings.Contains(name, "/") || !strings.HasSuffix(name, ".js") {
			http.NotFound(w, r)
			return
		}
		b, err := webAssets.ReadFile("webassets/games/common/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(b) // #nosec G705 - static JavaScript file, no user input
	})

	// Public.
	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/logout", s.handleLogout)
	mux.HandleFunc("/denied", s.handleDenied)

	// Authenticated pages.
	mux.HandleFunc("/", s.auth(s.handleArmory))
	mux.HandleFunc("/inventory", s.auth(s.handleInventory))
	mux.HandleFunc("/arcade", s.auth(s.handleArcadePage))
	mux.HandleFunc("/shop", s.auth(s.handleShopPage))
	mux.HandleFunc("/ah", s.auth(s.handleAHPage))
	mux.HandleFunc("/leaderboards", s.auth(s.handleLeaderboardsPage))

	if s.bot.Cfg.EnableAbyss {
		mux.HandleFunc("/abyss", s.auth(s.handleAbyssPage))
		mux.HandleFunc("/api/abyss/enter", s.authAPI(s.handleAbyssEnter))
		mux.HandleFunc("/api/abyss/descend", s.authAPI(s.handleAbyssDescend))
		mux.HandleFunc("/api/abyss/descend_multi", s.authAPI(s.handleAbyssDescendMulti))
		mux.HandleFunc("/api/abyss/choose_floor", s.authAPI(s.handleAbyssChooseFloor))
		mux.HandleFunc("/api/abyss/revive", s.authAPI(s.handleAbyssRevive))
		mux.HandleFunc("/api/abyss/concede", s.authAPI(s.handleAbyssConcede))
		mux.HandleFunc("/api/abyss/bank", s.authAPI(s.handleAbyssBank))
		mux.HandleFunc("/api/abyss/insure", s.authAPI(s.handleAbyssInsure))
		mux.HandleFunc("/api/abyss/salvage", s.authAPI(s.handleAbyssSalvage))
		mux.HandleFunc("/api/abyss/upgrade", s.authAPI(s.handleAbyssUpgrade))
		mux.HandleFunc("/api/abyss/use_consumable", s.authAPI(s.handleAbyssUseConsumable))
		mux.HandleFunc("/api/abyss/noncombat/action", s.authAPI(s.handleAbyssNonCombatAction))
		mux.HandleFunc("/api/abyss/noncombat/proceed", s.authAPI(s.handleAbyssNonCombatProceed))
		mux.HandleFunc("/api/abyss/coop/list", s.authAPI(s.handleAbyssCoopList))
		mux.HandleFunc("/api/abyss/coop/invite", s.authAPI(s.handleAbyssCoopInvite))
		mux.HandleFunc("/api/abyss/prestige", s.authAPI(s.handleAbyssPrestige))
		mux.HandleFunc("/api/abyss/bounty/claim", s.authAPI(s.handleAbyssBountyClaim))
		mux.HandleFunc("/api/abyss/set_badge", s.authAPI(s.handleAbyssSetBadge))
		mux.HandleFunc("/api/abyss/shop/buy", s.authAPI(s.handleAbyssShopBuy))
		mux.HandleFunc("/api/abyss/dismantle", s.authAPI(s.handleAbyssDismantle))
		mux.HandleFunc("/api/abyss/identify", s.authAPI(s.handleAbyssIdentify))
		mux.HandleFunc("/api/abyss/socket_gem", s.authAPI(s.handleAbyssSocketGem))
		mux.HandleFunc("/api/abyss/etch_rune", s.authAPI(s.handleAbyssEtchRune))
		mux.HandleFunc("/api/abyss/recalibrate", s.authAPI(s.handleAbyssRecalibrate))
		mux.HandleFunc("/api/abyss/upgrade_gear", s.authAPI(s.handleAbyssUpgradeGear))
		mux.HandleFunc("/api/abyss/transmute", s.authAPI(s.handleAbyssTransmute))
		mux.HandleFunc("/api/abyss/convert_mana", s.authAPI(s.handleAbyssConvertMana))
		mux.HandleFunc("/api/abyss/reset_talents", s.authAPI(s.handleAbyssResetTalents))
		mux.HandleFunc("/api/abyss/insure_item", s.authAPI(s.handleAbyssInsureItem))
		// Expansion 2 (docs/ABYSS_IDEAS.md)
		mux.HandleFunc("/api/abyss/craft", s.authAPI(s.handleAbyssCraft))
		mux.HandleFunc("/api/abyss/craft_legendary", s.authAPI(s.handleAbyssCraftLegendary))
		mux.HandleFunc("/api/abyss/exchange", s.authAPI(s.handleAbyssExchange))
		mux.HandleFunc("/api/abyss/temper", s.authAPI(s.handleAbyssTemper))
		mux.HandleFunc("/api/abyss/upgrade_gem", s.authAPI(s.handleAbyssUpgradeGem))
		mux.HandleFunc("/api/abyss/extract_gem", s.authAPI(s.handleAbyssExtractGem))
		mux.HandleFunc("/api/abyss/transfer_enchant", s.authAPI(s.handleAbyssTransferEnchant))
		mux.HandleFunc("/api/abyss/fuse", s.authAPI(s.handleAbyssFuse))
		mux.HandleFunc("/api/abyss/mythic_fuse", s.authAPI(s.handleAbyssMythicFuse))
		mux.HandleFunc("/api/abyss/cleanse", s.authAPI(s.handleAbyssCleanse))
		mux.HandleFunc("/api/abyss/repair_all", s.authAPI(s.handleAbyssRepairAll))
		mux.HandleFunc("/api/abyss/auto_repair", s.authAPI(s.handleAbyssAutoRepair))
		mux.HandleFunc("/api/abyss/identify_all", s.authAPI(s.handleAbyssIdentifyAll))
		mux.HandleFunc("/api/abyss/last_stand", s.authAPI(s.handleAbyssLastStand))
		mux.HandleFunc("/api/abyss/set_spec", s.authAPI(s.handleAbyssSetSpec))
		mux.HandleFunc("/api/abyss/sanctuary_buy", s.authAPI(s.handleAbyssSanctuaryBuy))
		mux.HandleFunc("/api/abyss/forge_history", s.authAPI(s.handleAbyssForgeHistory))
		mux.HandleFunc("/api/abyss/forge_undo", s.authAPI(s.handleAbyssForgeUndo))
		mux.HandleFunc("/api/abyss/rift_peek", s.authAPI(s.handleAbyssRiftPeek))
		mux.HandleFunc("/api/abyss/unequip", s.authAPI(s.handleAbyssUnequip))
		mux.HandleFunc("/abyss/tree", s.auth(s.handleAbyssTreePage))
		mux.HandleFunc("/api/abyss/tree/allocate", s.authAPI(s.handleAbyssTreeAllocate))
		mux.HandleFunc("/api/abyss/tree/respec", s.authAPI(s.handleAbyssTreeRespec))
		mux.HandleFunc("/api/abyss/tree/refund", s.authAPI(s.handleAbyssTreeRefund))
		mux.HandleFunc("/api/abyss/tree/socket", s.authAPI(s.handleAbyssTreeSocket))
		mux.HandleFunc("/api/abyss/tree/roll_timeless", s.authAPI(s.handleAbyssTreeRollTimeless))
		mux.HandleFunc("/api/abyss/tree/activate_keystone", s.authAPI(s.handleAbyssTreeActivateKeystone))
	}

	// Authenticated JSON APIs.
	mux.HandleFunc("/api/arcade/play", s.authAPI(s.handleArcadeAPI))
	mux.HandleFunc("/api/arcade/daily-spin", s.authAPI(s.handleDailySpinAPI))
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
	go func() { // #nosec G118 - graceful shutdown goroutine, acceptable to use background context
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
	msg := i18n.T("web.login_pm", url)
	if b.Cfg.EnableAbyss {
		var bestDepth int
		_ = b.DB.QueryRow("SELECT abyss_best_depth FROM users WHERE client_uid=$1", uid).Scan(&bestDepth)
		// Point at the tokenized login link (which authenticates) with a post-login
		// redirect to /abyss, instead of the bare protected route which would just
		// bounce a signed-out user to /denied.
		abyssURL := b.loginURL(token) + "&next=%2Fabyss"
		if short, err := games.ShortenURL(abyssURL); err == nil && short != "" {
			abyssURL = short
		}
		msg += fmt.Sprintf("\n⚔️ [b]The Abyss awaits![/b] Your best: floor %d.\nEnter the depths: %s", bestDepth, abyssURL)
	}
	return msg
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
	http.SetCookie(w, &http.Cookie{ // #nosec G124 - Secure flag is conditionally set based on HTTPS
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(90 * 24 * time.Hour),
	})
	// Honor an optional post-login destination, but only same-origin relative
	// paths to avoid an open redirect: must start with a single "/", must not be
	// scheme-relative ("//host") and must not start with "/\" (browsers
	// normalize backslashes to slashes, which would re-open the "//" bypass).
	dest := "/"
	if next := r.URL.Query().Get("next"); strings.HasPrefix(next, "/") &&
		!strings.HasPrefix(next, "//") && !strings.HasPrefix(next, "/\\") {
		if u, err := url.Parse(next); err == nil && u.Scheme == "" && u.Host == "" {
			dest = next
		}
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

func (s *WebServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	secure := strings.HasPrefix(strings.ToLower(s.bot.Cfg.WebBaseURL), "https://")
	http.SetCookie(w, &http.Cookie{ // #nosec G124 - Secure flag is conditionally set based on HTTPS
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/denied", http.StatusSeeOther)
}

func (s *WebServer) handleDenied(w http.ResponseWriter, _ *http.Request) {
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
	AbyssTokens int
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
		"SELECT nickname, level, prestige, xp, gold, abyss_tokens, current_hp FROM users WHERE client_uid=$1", uid,
	).Scan(&nick, &u.Level, &u.Prestige, &u.XP, &u.Gold, &u.AbyssTokens, &u.CurrentHP)
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
	// Surface the Abyss feature flag to every page so the shared top-nav can hide
	// the Abyss link wherever its routes aren't registered.
	if m, ok := data.(map[string]any); ok {
		if _, exists := m["EnableAbyss"]; !exists {
			m["EnableAbyss"] = s.bot.Cfg.EnableAbyss
		}
	}
	// Render into a buffer first. Executing straight into the ResponseWriter means a
	// mid-template error (e.g. a struct missing a field the template references) flushes
	// a half-written page — truncating the trailing <script> so every inline handler is
	// "not defined". Buffering lets us send a clean 500 instead of a broken page.
	var buf bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("web: render %s failed: %v", name, err)
		http.Error(w, "render error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}
