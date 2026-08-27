package bot

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	abyssCoreActionMaxBytes        = 256 * 1024
	abyssCoreActionCacheTTL        = 10 * time.Minute
	abyssCoreActionCacheLimit      = 64
	abyssCoreActionRateBurst       = 8
	abyssCoreActionRatePerSecond   = 2.0
	abyssCoreActionRateStateTTL    = 15 * time.Minute
	abyssCoreActionReplayHeader    = "X-Idempotent-Replay"
	abyssCoreActionRequestIDHeader = "Idempotency-Key"
)

type abyssCoreActionHandler func(http.ResponseWriter, *http.Request, string)

type abyssCoreActionGuard struct {
	mu          sync.Mutex
	now         func() time.Time
	locks       map[string]*abyssCoreActionLock
	cache       map[string]abyssCoreActionCachedResponse
	rates       map[string]abyssCoreActionRateState
	nextCleanup time.Time
}

type abyssCoreActionLock struct {
	mu   sync.Mutex
	refs int
}

type abyssCoreActionCachedResponse struct {
	uid         string
	requestHash [sha256.Size]byte
	status      int
	header      http.Header
	body        []byte
	createdAt   time.Time
	expiresAt   time.Time
}

type abyssCoreActionRateState struct {
	tokens       float64
	updated      time.Time
	lastSeen     time.Time
	deniedSince  time.Time
	deniedCount  int
	lastMacroLog time.Time
}

type abyssCoreActionResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newAbyssCoreActionResponse() *abyssCoreActionResponse {
	return &abyssCoreActionResponse{header: make(http.Header)}
}

func (w *abyssCoreActionResponse) Header() http.Header { return w.header }

func (w *abyssCoreActionResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *abyssCoreActionResponse) Write(payload []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(payload)
}

func (w *abyssCoreActionResponse) successfulJSON() bool {
	var result struct {
		OK bool `json:"ok"`
	}
	return w.status >= 200 && w.status < 300 && json.Unmarshal(w.body.Bytes(), &result) == nil && result.OK
}

func (s *WebServer) guardAbyssCoreAction(next abyssCoreActionHandler) abyssCoreActionHandler {
	return func(w http.ResponseWriter, r *http.Request, uid string) {
		key := r.Header.Get(abyssCoreActionRequestIDHeader)
		if key == "" {
			if retryAfter, macro := s.abyssCoreActions.takeRateToken(uid); retryAfter > 0 {
				if macro {
					log.Printf("web: repeated Abyss action burst detected uid=%q path=%q", uid, r.URL.Path)
				}
				writeAbyssCoreActionRateLimit(w, retryAfter)
				return
			}
			next(w, r, uid)
			return
		}
		if !validAbyssCoreActionKey(key) {
			writeJSONStatus(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid idempotency key"})
			return
		}

		payload, err := readAbyssCoreActionBody(r.Body)
		if err != nil {
			writeJSONStatus(w, http.StatusRequestEntityTooLarge, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(payload))
		requestHash := sha256.Sum256(payload)
		cacheKey := uid + "\x00" + r.URL.Path + "\x00" + key
		release := s.abyssCoreActions.acquire(cacheKey)
		defer release()

		if cached, ok := s.abyssCoreActions.cached(cacheKey); ok {
			if cached.requestHash != requestHash {
				writeJSONStatus(w, http.StatusConflict, map[string]any{"ok": false, "error": "idempotency key was already used with another payload"})
				return
			}
			writeAbyssCoreActionCachedResponse(w, cached)
			return
		}
		if retryAfter, macro := s.abyssCoreActions.takeRateToken(uid); retryAfter > 0 {
			if macro {
				log.Printf("web: repeated Abyss action burst detected uid=%q path=%q", uid, r.URL.Path)
			}
			writeAbyssCoreActionRateLimit(w, retryAfter)
			return
		}

		response := newAbyssCoreActionResponse()
		next(response, r, uid)
		if response.successfulJSON() {
			s.abyssCoreActions.store(cacheKey, uid, requestHash, response)
		}
		writeAbyssCoreActionResponse(w, response)
	}
}

func validAbyssCoreActionKey(key string) bool {
	if len(key) < 8 || len(key) > 128 {
		return false
	}
	for _, char := range key {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' || char == ':' {
			continue
		}
		return false
	}
	return true
}

func readAbyssCoreActionBody(body io.Reader) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	payload, err := io.ReadAll(io.LimitReader(body, abyssCoreActionMaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read Abyss action: %w", err)
	}
	if len(payload) > abyssCoreActionMaxBytes {
		return nil, fmt.Errorf("Abyss action exceeds %d bytes", abyssCoreActionMaxBytes)
	}
	return payload, nil
}

func (g *abyssCoreActionGuard) acquire(key string) func() {
	g.mu.Lock()
	if g.locks == nil {
		g.locks = make(map[string]*abyssCoreActionLock)
	}
	lock := g.locks[key]
	if lock == nil {
		lock = &abyssCoreActionLock{}
		g.locks[key] = lock
	}
	lock.refs++
	g.mu.Unlock()

	lock.mu.Lock()
	return func() {
		lock.mu.Unlock()
		g.mu.Lock()
		lock.refs--
		if lock.refs == 0 {
			delete(g.locks, key)
		}
		g.mu.Unlock()
	}
}

func (g *abyssCoreActionGuard) cached(key string) (abyssCoreActionCachedResponse, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.timeLocked()
	cached, ok := g.cache[key]
	if ok && !now.Before(cached.expiresAt) {
		delete(g.cache, key)
		return abyssCoreActionCachedResponse{}, false
	}
	return cached, ok
}

func (g *abyssCoreActionGuard) store(key, uid string, requestHash [sha256.Size]byte, response *abyssCoreActionResponse) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.timeLocked()
	if g.cache == nil {
		g.cache = make(map[string]abyssCoreActionCachedResponse)
	}
	for cacheKey, cached := range g.cache {
		if !now.Before(cached.expiresAt) {
			delete(g.cache, cacheKey)
		}
	}

	status := response.status
	if status == 0 {
		status = http.StatusOK
	}
	g.cache[key] = abyssCoreActionCachedResponse{
		uid: uid, requestHash: requestHash, status: status, header: response.header.Clone(),
		body: append([]byte(nil), response.body.Bytes()...), createdAt: now, expiresAt: now.Add(abyssCoreActionCacheTTL),
	}

	for g.cacheCountLocked(uid) > abyssCoreActionCacheLimit {
		oldestKey := ""
		var oldest time.Time
		for cacheKey, cached := range g.cache {
			if cached.uid == uid && (oldestKey == "" || cached.createdAt.Before(oldest)) {
				oldestKey, oldest = cacheKey, cached.createdAt
			}
		}
		delete(g.cache, oldestKey)
	}
}

func (g *abyssCoreActionGuard) cacheCountLocked(uid string) int {
	count := 0
	for _, cached := range g.cache {
		if cached.uid == uid {
			count++
		}
	}
	return count
}

func (g *abyssCoreActionGuard) takeRateToken(uid string) (time.Duration, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.timeLocked()
	if g.rates == nil {
		g.rates = make(map[string]abyssCoreActionRateState)
	}
	if g.nextCleanup.IsZero() || !now.Before(g.nextCleanup) {
		for rateUID, state := range g.rates {
			if now.Sub(state.lastSeen) >= abyssCoreActionRateStateTTL {
				delete(g.rates, rateUID)
			}
		}
		g.nextCleanup = now.Add(abyssCoreActionRateStateTTL)
	}

	state, ok := g.rates[uid]
	if !ok {
		state = abyssCoreActionRateState{tokens: abyssCoreActionRateBurst, updated: now}
	}
	elapsed := now.Sub(state.updated).Seconds()
	if elapsed > 0 {
		state.tokens = math.Min(abyssCoreActionRateBurst, state.tokens+elapsed*abyssCoreActionRatePerSecond)
	}
	state.updated = now
	state.lastSeen = now
	if state.tokens >= 1 {
		state.tokens--
		state.deniedSince = time.Time{}
		state.deniedCount = 0
		g.rates[uid] = state
		return 0, false
	}
	if state.deniedSince.IsZero() || now.Sub(state.deniedSince) > 2*time.Second {
		state.deniedSince = now
		state.deniedCount = 1
	} else {
		state.deniedCount++
	}
	macro := state.deniedCount >= 3 && (state.lastMacroLog.IsZero() || now.Sub(state.lastMacroLog) >= time.Minute)
	if macro {
		state.lastMacroLog = now
	}
	g.rates[uid] = state
	retryAfter := time.Duration(math.Ceil((1-state.tokens)/abyssCoreActionRatePerSecond*1000)) * time.Millisecond
	return retryAfter, macro
}

func (g *abyssCoreActionGuard) timeLocked() time.Time {
	if g.now != nil {
		return g.now()
	}
	return time.Now()
}

func writeAbyssCoreActionRateLimit(w http.ResponseWriter, retryAfter time.Duration) {
	milliseconds := max(1, int(math.Ceil(float64(retryAfter)/float64(time.Millisecond))))
	seconds := max(1, int(math.Ceil(float64(retryAfter)/float64(time.Second))))
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	writeJSONStatus(w, http.StatusTooManyRequests, map[string]any{
		"ok": false, "error": "too many Abyss actions — wait a moment", "retry_after_ms": milliseconds,
	})
}

func writeAbyssCoreActionCachedResponse(w http.ResponseWriter, cached abyssCoreActionCachedResponse) {
	for key, values := range cached.header {
		w.Header()[key] = append([]string(nil), values...)
	}
	w.Header().Set(abyssCoreActionReplayHeader, "true")
	w.WriteHeader(cached.status)
	_, _ = w.Write(cached.body)
}

func writeAbyssCoreActionResponse(w http.ResponseWriter, response *abyssCoreActionResponse) {
	for key, values := range response.header {
		w.Header()[key] = append([]string(nil), values...)
	}
	status := response.status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_, _ = w.Write(response.body.Bytes())
}
