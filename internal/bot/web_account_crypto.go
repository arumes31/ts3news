package bot

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"
)

const accountPasswordIterations = 600000
const accountCSRFCookie = "ts3csrf"
const accountRecoveryCookie = "ts3recovery"

// PBKDF2-HMAC-SHA256 uses the OWASP work factor and Go's standard implementation.
// https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
func hashAccountPassword(password string) (string, error) {
	if !utf8.ValidString(password) || utf8.RuneCountInString(password) < 15 || utf8.RuneCountInString(password) > 128 || len(password) > 512 {
		return "", errors.New("use a password between 15 and 128 characters")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, accountPasswordIterations, 32)
	if err != nil {
		return "", err
	}
	return "pbkdf2-sha256$600000$" + hex.EncodeToString(salt) + "$" + hex.EncodeToString(key), nil
}

func verifyAccountPassword(encoded, password string) bool {
	if len(password) > 512 {
		return false
	}
	parts := strings.Split(encoded, "$")
	valid := len(parts) == 4 && parts[0] == "pbkdf2-sha256" && parts[1] == "600000"
	salt, want := make([]byte, 16), make([]byte, 32)
	if valid {
		var err error
		salt, err = hex.DecodeString(parts[2])
		valid = err == nil && len(salt) == 16
		want, err = hex.DecodeString(parts[3])
		valid = valid && err == nil && len(want) == 32
	}
	if !valid {
		salt, want = make([]byte, 16), make([]byte, 32)
	}
	// Unknown usernames and accounts without a password still do the same work.
	got, err := pbkdf2.Key(sha256.New, password, salt, accountPasswordIterations, 32)
	return err == nil && subtle.ConstantTimeCompare(got, want) == 1 && valid
}

func newAccountToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func accountDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func accountCookieValue(r *http.Request, name string) string {
	if cookie, err := r.Cookie(name); err == nil {
		return cookie.Value
	}
	return ""
}

func (s *WebServer) accountSecure() bool {
	return s.bot != nil && s.bot.Cfg != nil && strings.HasPrefix(strings.ToLower(s.bot.Cfg.WebBaseURL), "https://")
}

func (s *WebServer) accountCSRFSignature(nonce string, r *http.Request) string {
	mac := hmac.New(sha256.New, s.forgeQuoteKey[:])
	_, _ = mac.Write([]byte(nonce + ":" + accountCookieValue(r, sessionCookie) + ":" + accountCookieValue(r, accountRecoveryCookie)))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *WebServer) accountCSRF(w http.ResponseWriter, r *http.Request) string {
	nonce, err := newAccountToken()
	if err != nil {
		return ""
	}
	token := nonce + "." + s.accountCSRFSignature(nonce, r)
	http.SetCookie(w, &http.Cookie{Name: accountCSRFCookie, Value: token, Path: "/", HttpOnly: true, Secure: s.accountSecure(), SameSite: http.SameSiteStrictMode})
	return token
}

func (s *WebServer) validAccountForm(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || !strings.EqualFold(u.Host, r.Host) || (s.accountSecure() && u.Scheme != "https") {
			return false
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if r.ParseForm() != nil {
		return false
	}
	token := r.PostForm.Get("csrf")
	nonce, signature, ok := strings.Cut(token, ".")
	return ok && len(nonce) == 64 && hmac.Equal([]byte(token), []byte(accountCookieValue(r, accountCSRFCookie))) && hmac.Equal([]byte(signature), []byte(s.accountCSRFSignature(nonce, r)))
}

func accountDestination(next string) string {
	u, err := url.Parse(next)
	if err != nil || u.IsAbs() || u.Host != "" || !strings.HasPrefix(u.Path, "/") || strings.HasPrefix(u.Path, "//") || strings.ContainsAny(u.Path, "\\\r\n") {
		return "/"
	}
	// Rebuild from local components only. Never carry an authority, scheme or
	// opaque URL from the caller into a Location header.
	local := &url.URL{Path: u.Path, RawQuery: u.RawQuery, ForceQuery: u.ForceQuery, Fragment: u.Fragment}
	return local.String()
}
