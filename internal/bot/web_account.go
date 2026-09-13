package bot

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
)

func accountHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func (s *WebServer) accountError(w http.ResponseWriter, err error) {
	// Avoid logging driver error text: it can contain credential query parameters.
	log.Printf("web account operation failed (%T)", err)
	http.Error(w, "Account service is unavailable. Please try again.", http.StatusServiceUnavailable)
}

func (s *WebServer) renderAccount(w http.ResponseWriter, r *http.Request, name string, data map[string]any) {
	accountHeaders(w)
	// Chrome sends Origin: null on form POSTs from no-referrer documents.
	// Forms need a same-origin policy; credential-exchange redirects above retain
	// no-referrer, and cross-origin destinations never receive a referrer.
	w.Header().Set("Referrer-Policy", "same-origin")
	data["CSRF"] = s.accountCSRF(w, r)
	data["AccountPage"] = true
	s.render(w, name, data)
}

func (s *WebServer) handleAccountLogin(w http.ResponseWriter, r *http.Request) {
	accountHeaders(w)
	data := map[string]any{"Title": "Sign in", "Next": accountDestination(r.URL.Query().Get("next"))}
	if r.Method == http.MethodGet {
		if token := r.URL.Query().Get("token"); token != "" {
			err := s.finishAccountLogin(w, r, "", token, false)
			if err == nil {
				http.Redirect(w, r, data["Next"].(string), http.StatusSeeOther)
				return
			}
			if !errors.Is(err, sql.ErrNoRows) {
				s.accountError(w, err)
				return
			}
			data["Error"] = "This TeamSpeak link has expired. Use your password or request a new link from the bot."
		}
		s.renderAccount(w, r, "account-login", data)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
		return
	}
	if !s.validAccountForm(w, r) {
		http.Error(w, "This form expired. Reload the page and try again.", http.StatusForbidden)
		return
	}
	username := strings.ToLower(strings.TrimSpace(r.PostForm.Get("username")))
	password := r.PostForm.Get("password")
	data["Username"], data["Next"] = username, accountDestination(r.PostForm.Get("next"))
	if !s.accountThrottle(w, r, "login:"+username) || !accountHashSlot(w) {
		return
	}
	defer func() { <-accountHashSlots }()
	var uid, hash string
	err := s.bot.DB.QueryRowContext(r.Context(), "SELECT client_uid, COALESCE(web_password_hash, '') FROM users WHERE lower(web_username)=$1", username).Scan(&uid, &hash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.accountError(w, err)
		return
	}
	if !verifyAccountPassword(hash, password) {
		data["Error"] = "Username or password is incorrect."
		s.renderAccount(w, r, "account-login", data)
		return
	}
	if err := s.finishAccountLogin(w, r, uid, hash, true); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			data["Error"] = "Your credentials changed. Please sign in again."
			s.renderAccount(w, r, "account-login", data)
			return
		}
		s.accountError(w, err)
		return
	}
	http.Redirect(w, r, data["Next"].(string), http.StatusSeeOther)
}

func (s *WebServer) handleSettings(w http.ResponseWriter, r *http.Request) {
	accountHeaders(w)
	uid, ok := s.accountSessionUID(r.Context(), accountCookieValue(r, sessionCookie))
	if !ok {
		http.Redirect(w, r, "/login?next=%2Fsettings", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
		return
	}
	var username, nickname, hash string
	err := s.bot.DB.QueryRowContext(r.Context(), "SELECT web_username, COALESCE(nickname, ''), COALESCE(web_password_hash, '') FROM users WHERE client_uid=$1", uid).Scan(&username, &nickname, &hash)
	if err != nil {
		s.accountError(w, err)
		return
	}
	data := map[string]any{"Title": "Settings", "Nav": "settings", "Username": username, "Nickname": nickname, "HasPassword": hash != "", "AccountNav": true, "Saved": r.URL.Query().Get("saved") == "1"}
	if r.Method == http.MethodPost {
		if !s.validAccountForm(w, r) {
			http.Error(w, "This form expired. Reload the page and try again.", http.StatusForbidden)
			return
		}
		if !s.accountThrottle(w, r, "change:"+uid) || !accountHashSlot(w) {
			return
		}
		defer func() { <-accountHashSlots }()
		if hash == "" {
			data["Error"] = "Verify your TeamSpeak identity with !password to set your first password."
		} else if !verifyAccountPassword(hash, r.PostForm.Get("current_password")) {
			data["Error"] = "Current password is incorrect."
		} else if r.PostForm.Get("password") != r.PostForm.Get("confirm_password") {
			data["Error"] = "The new passwords do not match."
		} else {
			newHash, hashErr := hashAccountPassword(r.PostForm.Get("password"))
			if hashErr != nil {
				data["Error"] = hashErr.Error()
			} else {
				if err := s.saveAccountPassword(w, r, uid, hash, newHash, ""); err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						data["Error"] = "Your session or password changed. Sign in again."
					} else {
						s.accountError(w, err)
						return
					}
				} else {
					http.Redirect(w, r, "/settings?saved=1", http.StatusSeeOther)
					return
				}
			}
		}
	}
	s.renderAccount(w, r, "account-settings", data)
}

func (s *WebServer) handleAccountRecovery(w http.ResponseWriter, r *http.Request) {
	accountHeaders(w)
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	if token := r.URL.Query().Get("token"); len(token) == 64 {
		// Strip the credential from the address bar before rendering any page or assets.
		// Lax permits the top-level redirect when the link is opened outside the
		// portal. Strict drops this cookie on that redirect; POSTs remain CSRF-bound.
		http.SetCookie(w, &http.Cookie{Name: accountRecoveryCookie, Value: token, Path: "/account/password", HttpOnly: true, Secure: s.accountSecure(), SameSite: http.SameSiteLaxMode, MaxAge: 600})
		http.Redirect(w, r, "/account/password", http.StatusSeeOther)
		return
	}
	s.renderAccount(w, r, "account-recovery", map[string]any{"Title": "TeamSpeak recovery"})
}

func (s *WebServer) handleAccountPassword(w http.ResponseWriter, r *http.Request) {
	accountHeaders(w)
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
		return
	}
	recovery := accountCookieValue(r, accountRecoveryCookie)
	var username string
	err := s.bot.DB.QueryRowContext(r.Context(), "SELECT web_username FROM users WHERE web_recovery_hash=$1 AND web_recovery_expires > NOW()", accountDigest(recovery)).Scan(&username)
	data := map[string]any{"Title": "Set password", "Username": username, "ValidRecovery": err == nil && recovery != ""}
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.accountError(w, err)
			return
		}
		data["Error"] = "This link has expired or was already used. Request a new one in TeamSpeak."
	}
	if r.Method == http.MethodPost {
		if !s.validAccountForm(w, r) {
			http.Error(w, "This form expired. Reload the page and try again.", http.StatusForbidden)
			return
		}
		if err != nil {
			s.renderAccount(w, r, "account-password", data)
			return
		}
		if !s.accountThrottle(w, r, "recovery:"+accountDigest(recovery)) || !accountHashSlot(w) {
			return
		}
		defer func() { <-accountHashSlots }()
		if r.PostForm.Get("password") != r.PostForm.Get("confirm_password") {
			data["Error"] = "The new passwords do not match."
		} else {
			hash, hashErr := hashAccountPassword(r.PostForm.Get("password"))
			if hashErr != nil {
				data["Error"] = hashErr.Error()
			} else {
				if err := s.saveAccountPassword(w, r, "", "", hash, recovery); err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						data["Error"], data["ValidRecovery"] = "This link has expired or was already used.", false
					} else {
						s.accountError(w, err)
						return
					}
				} else {
					http.SetCookie(w, &http.Cookie{Name: accountRecoveryCookie, Value: "", Path: "/account/password", HttpOnly: true, Secure: s.accountSecure(), SameSite: http.SameSiteLaxMode, MaxAge: -1})
					http.Redirect(w, r, "/settings?saved=1", http.StatusSeeOther)
					return
				}
			}
		}
	}
	s.renderAccount(w, r, "account-password", data)
}

// All credential changes and revocations commit together. The users row lock
// serializes these operations with both login paths and recovery issuance.
func (s *WebServer) saveAccountPassword(w http.ResponseWriter, r *http.Request, uid, oldHash, newHash, recovery string) error {
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if recovery != "" {
		err = tx.QueryRowContext(r.Context(), `UPDATE users SET web_password_hash=$1, web_recovery_hash=NULL,
 web_recovery_expires=NULL, web_token=NULL, web_token_expires=NULL
 WHERE web_recovery_hash=$2 AND web_recovery_expires > NOW() RETURNING client_uid`, newHash, accountDigest(recovery)).Scan(&uid)
	} else {
		// Lock first, then re-check the session in a new statement snapshot.
		var found string
		err = tx.QueryRowContext(r.Context(), "SELECT client_uid FROM users WHERE client_uid=$1 AND web_password_hash=$2 FOR UPDATE", uid, oldHash).Scan(&found)
		if err == nil {
			err = tx.QueryRowContext(r.Context(), "SELECT client_uid FROM web_sessions WHERE token_hash=$1 AND client_uid=$2 AND expires_at > NOW()", accountDigest(accountCookieValue(r, sessionCookie)), uid).Scan(&found)
		}
		if err == nil {
			_, err = tx.ExecContext(r.Context(), `UPDATE users SET web_password_hash=$1, web_recovery_hash=NULL,
 web_recovery_expires=NULL, web_token=NULL, web_token_expires=NULL WHERE client_uid=$2`, newHash, uid)
		}
	}
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(r.Context(), "DELETE FROM web_sessions WHERE client_uid=$1", uid); err != nil {
		return err
	}
	token, expires, err := s.createAccountSession(r.Context(), tx, uid)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	s.setAccountSession(w, token, expires)
	log.Print("web account password updated; previous sessions revoked")
	return nil
}

func (s *WebServer) runAccountCleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Cleanup is started by Start, with its own bounded database context.
			cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			if err := s.cleanupAccountAuth(cleanupCtx); err != nil {
				log.Printf("web account cleanup failed (%T)", err)
			}
			cancel()
		}
	}
}
