package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

var accountHashSlots = make(chan struct{}, 4)

func accountHashSlot(w http.ResponseWriter) bool {
	select {
	case accountHashSlots <- struct{}{}:
		return true
	default:
		w.Header().Set("Retry-After", "5")
		http.Error(w, "Sign-in is busy. Try again shortly.", http.StatusTooManyRequests)
		return false
	}
}

func (s *WebServer) accountAttempt(ctx context.Context, key string, limit int) (bool, error) {
	var count int
	err := s.bot.DB.QueryRowContext(ctx, `INSERT INTO web_auth_attempts (key_hash, attempts, expires_at)
 VALUES ($1, 1, NOW() + INTERVAL '15 minutes')
 ON CONFLICT (key_hash) DO UPDATE SET
 attempts = CASE WHEN web_auth_attempts.expires_at <= NOW() THEN 1 ELSE LEAST(web_auth_attempts.attempts + 1, 10000) END,
 expires_at = CASE WHEN web_auth_attempts.expires_at <= NOW() THEN NOW() + INTERVAL '15 minutes' ELSE web_auth_attempts.expires_at END
 RETURNING attempts`, accountDigest(key)).Scan(&count)
	return count <= limit, err
}

func (s *WebServer) accountThrottle(w http.ResponseWriter, r *http.Request, key string) bool {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	// Forwarding headers are intentionally not trusted without a proxy allowlist.
	purpose, _, _ := strings.Cut(key, ":")
	for _, entry := range []struct {
		key   string
		limit int
	}{{"ip:" + purpose + ":" + ip, 60}, {"account:" + key, 10}} {
		ok, err := s.accountAttempt(r.Context(), entry.key, entry.limit)
		if err != nil {
			s.accountError(w, err)
			return false
		}
		if !ok {
			w.Header().Set("Retry-After", "900")
			http.Error(w, "Too many attempts. Try again in 15 minutes.", http.StatusTooManyRequests)
			return false
		}
	}
	return true
}

func (s *WebServer) createAccountSession(ctx context.Context, tx *sql.Tx, uid string) (string, time.Time, error) {
	token, err := newAccountToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().Add(sessionLifetime)
	// The caller holds the identity row lock. Keep at most 20 device sessions,
	// including the one about to be created, even when a reusable link is replayed.
	if _, err := tx.ExecContext(ctx, `DELETE FROM web_sessions WHERE token_hash IN
 (SELECT token_hash FROM web_sessions WHERE client_uid=$1 ORDER BY expires_at DESC, token_hash OFFSET 19)`, uid); err != nil {
		return "", time.Time{}, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO web_sessions (token_hash, client_uid, expires_at) VALUES ($1, $2, $3)", accountDigest(token), uid, expires)
	return token, expires, err
}

func (s *WebServer) setAccountSession(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: s.accountSecure(), SameSite: http.SameSiteLaxMode, Expires: expires})
	http.SetCookie(w, &http.Cookie{Name: sessionExpiryCookie, Value: fmt.Sprintf("%d", expires.Unix()), Path: "/", Secure: s.accountSecure(), SameSite: http.SameSiteLaxMode, Expires: expires})
}

// finishAccountLogin locks the identity until session issuance commits. A concurrent
// password reset cannot validate an old credential and then issue a new session.
func (s *WebServer) finishAccountLogin(w http.ResponseWriter, r *http.Request, uid, credential string, password bool) error {
	if !password && (credential == "" || len(credential) > 128) {
		return sql.ErrNoRows
	}
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var found string
	if password {
		err = tx.QueryRowContext(r.Context(), "SELECT client_uid FROM users WHERE client_uid=$1 AND web_password_hash=$2 FOR UPDATE", uid, credential).Scan(&found)
	} else {
		err = tx.QueryRowContext(r.Context(), "SELECT client_uid FROM users WHERE web_token=$1 AND (web_token_expires IS NULL OR web_token_expires > NOW()) FOR UPDATE", credential).Scan(&found)
	}
	if err != nil {
		return err
	}
	token, expires, err := s.createAccountSession(r.Context(), tx, found)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.setAccountSession(w, token, expires)
	return nil
}

func (s *WebServer) revokeAccountSession(ctx context.Context, token string) error {
	tx, err := s.bot.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Only migrated cookies equal a TeamSpeak link credential. Revoke that link
	// too, otherwise the logged-out cookie could be exchanged for a new session.
	if _, err := tx.ExecContext(ctx, "/* economy:bot.WebServer.revokeAccountSession */ UPDATE users SET web_token=NULL, web_token_expires=NULL WHERE web_token=$1", token); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM web_sessions WHERE token_hash=$1", accountDigest(token)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *WebServer) accountSessionUID(ctx context.Context, token string) (string, bool) {
	if token == "" || len(token) > 128 {
		return "", false
	}
	var uid string
	err := s.bot.DB.QueryRowContext(ctx, "SELECT client_uid FROM web_sessions WHERE token_hash=$1 AND expires_at > NOW()", accountDigest(token)).Scan(&uid)
	return uid, err == nil
}

// issueAccountRecovery is called only after the ClientQuery sender UID is verified.
func (b *Bot) issueAccountRecovery(ctx context.Context, uid string) (string, error) {
	token, err := newAccountToken()
	if err != nil {
		return "", err
	}
	result, err := b.DB.ExecContext(ctx, `/* economy:bot.Bot.issueAccountRecovery */ UPDATE users SET web_recovery_hash=$1,
 web_recovery_expires=NOW() + INTERVAL '10 minutes', web_recovery_issued=NOW()
 WHERE client_uid=$2 AND (web_recovery_issued IS NULL OR web_recovery_issued < NOW() - INTERVAL '1 minute')`, accountDigest(token), uid)
	if err != nil {
		return "", err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return "", err
	}
	if count != 1 {
		return "", errors.New("recovery unavailable or requested too recently")
	}
	return token, nil
}

func (s *WebServer) cleanupAccountAuth(ctx context.Context) error {
	if _, err := s.bot.DB.ExecContext(ctx, "DELETE FROM web_sessions WHERE expires_at <= NOW()"); err != nil {
		return err
	}
	_, err := s.bot.DB.ExecContext(ctx, "DELETE FROM web_auth_attempts WHERE expires_at <= NOW()")
	return err
}
