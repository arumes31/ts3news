package bot

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	abyssAPITokenPrefix = "abp_"
	abyssAPITokenBytes  = 32
	abyssAPITokenMaxLen = 128
)

type abyssAPITokenStatus struct {
	Configured bool   `json:"configured"`
	Prefix     string `json:"prefix,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
}

func newAbyssAPIToken() (string, []byte, string, error) {
	raw := make([]byte, abyssAPITokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, "", fmt.Errorf("generating API token: %w", err)
	}
	token := abyssAPITokenPrefix + base64.RawURLEncoding.EncodeToString(raw)
	digest := sha256.Sum256([]byte(token))
	return token, digest[:], token[:12], nil
}

func (s *WebServer) handleAbyssAPIToken(
	w http.ResponseWriter,
	r *http.Request,
	uid string,
) {
	w.Header().Set("Cache-Control", "private, no-store")
	switch r.Method {
	case http.MethodGet:
		status, err := s.abyssAPITokenStatus(r.Context(), uid)
		if err != nil {
			writeJSONStatus(w, http.StatusServiceUnavailable, map[string]any{
				"ok": false, "error": "API token unavailable",
			})
			return
		}
		writeJSON(w, map[string]any{"ok": true, "token": status})
	case http.MethodPost:
		if !strings.HasPrefix(
			strings.ToLower(r.Header.Get("Content-Type")),
			"application/json",
		) {
			writeJSONStatus(w, http.StatusUnsupportedMediaType, map[string]any{
				"ok": false, "error": "application/json required",
			})
			return
		}
		token, hash, prefix, err := newAbyssAPIToken()
		if err != nil {
			writeJSONStatus(w, http.StatusServiceUnavailable, map[string]any{
				"ok": false, "error": "API token unavailable",
			})
			return
		}
		if _, err := s.bot.DB.ExecContext(
			r.Context(),
			`INSERT INTO abyss_api_tokens (client_uid,token_prefix,token_hash)
			 VALUES ($1,$2,$3)
			 ON CONFLICT (client_uid) DO UPDATE SET
			 token_prefix=EXCLUDED.token_prefix,token_hash=EXCLUDED.token_hash,created_at=NOW()`,
			uid,
			prefix,
			hash,
		); err != nil {
			writeJSONStatus(w, http.StatusServiceUnavailable, map[string]any{
				"ok": false, "error": "API token unavailable",
			})
			return
		}
		writeJSON(w, map[string]any{
			"ok": true, "token": token, "prefix": prefix,
			"notice": "Copy this token now. It is not shown again.",
		})
	case http.MethodDelete:
		if _, err := s.bot.DB.ExecContext(
			r.Context(),
			"DELETE FROM abyss_api_tokens WHERE client_uid=$1",
			uid,
		); err != nil {
			writeJSONStatus(w, http.StatusServiceUnavailable, map[string]any{
				"ok": false, "error": "API token unavailable",
			})
			return
		}
		writeJSON(w, map[string]any{"ok": true, "revoked": true})
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		writeJSONStatus(w, http.StatusMethodNotAllowed, map[string]any{
			"ok": false, "error": "GET, POST, or DELETE only",
		})
	}
}

func (s *WebServer) abyssAPITokenStatus(
	ctx context.Context,
	uid string,
) (abyssAPITokenStatus, error) {
	var status abyssAPITokenStatus
	var created time.Time
	err := s.bot.DB.QueryRowContext(
		ctx,
		"SELECT token_prefix,created_at FROM abyss_api_tokens WHERE client_uid=$1",
		uid,
	).Scan(&status.Prefix, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return status, nil
	}
	if err != nil {
		return abyssAPITokenStatus{}, fmt.Errorf("loading API token status: %w", err)
	}
	status.Configured = true
	status.CreatedAt = created.UTC().Format(time.RFC3339)
	return status, nil
}

func (s *WebServer) handleAbyssTokenStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Vary", "Authorization")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONStatus(w, http.StatusMethodNotAllowed, map[string]any{
			"ok": false, "error": "GET only",
		})
		return
	}
	if err := s.authenticateAbyssAPIToken(
		r.Context(),
		r.Header.Get("Authorization"),
	); err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer realm="Abyss public stats"`)
		writeJSONStatus(w, http.StatusUnauthorized, map[string]any{
			"ok": false, "error": "invalid API token",
		})
		return
	}
	snapshot, err := s.loadAbyssPublicStats(r.Context(), time.Now().UTC())
	if err != nil {
		writeJSONStatus(w, http.StatusServiceUnavailable, map[string]any{
			"ok": false, "error": "Abyss stats are temporarily unavailable",
		})
		return
	}
	writeJSON(w, snapshot)
}

func (s *WebServer) authenticateAbyssAPIToken(
	ctx context.Context,
	authorization string,
) error {
	parts := strings.Fields(authorization)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return errors.New("missing bearer token")
	}
	token := parts[1]
	if len(token) < 12 || len(token) > abyssAPITokenMaxLen ||
		!strings.HasPrefix(token, abyssAPITokenPrefix) {
		return errors.New("invalid bearer token")
	}
	var storedHash []byte
	if err := s.bot.DB.QueryRowContext(
		ctx,
		"SELECT token_hash FROM abyss_api_tokens WHERE token_prefix=$1",
		token[:12],
	).Scan(&storedHash); err != nil {
		return errors.New("invalid bearer token")
	}
	digest := sha256.Sum256([]byte(token))
	if len(storedHash) != len(digest) ||
		subtle.ConstantTimeCompare(storedHash, digest[:]) != 1 {
		return errors.New("invalid bearer token")
	}
	return nil
}
