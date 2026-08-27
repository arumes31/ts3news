package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAbyssCoreActionGuardReplaysConcurrentRequest(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	server := &WebServer{}
	handler := server.guardAbyssCoreAction(func(w http.ResponseWriter, _ *http.Request, _ string) {
		calls.Add(1)
		entered <- struct{}{}
		<-release
		writeJSON(w, map[string]any{"ok": true, "depth": 12})
	})

	responses := make(chan *httptest.ResponseRecorder, 2)
	invoke := func() {
		request := newAbyssCoreActionRequest("retry-key-123", []byte(`{"interactive":true}`))
		response := httptest.NewRecorder()
		handler(response, request, "player-1")
		responses <- response
	}
	go invoke()
	<-entered
	go invoke()
	close(release)

	first, second := <-responses, <-responses
	if got := calls.Load(); got != 1 {
		t.Fatalf("handler calls = %d, want 1", got)
	}
	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("response status = %d, %d; want 200, 200", first.Code, second.Code)
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("response bodies differ: %q != %q", first.Body.String(), second.Body.String())
	}
	if first.Header().Get(abyssCoreActionReplayHeader) != "true" && second.Header().Get(abyssCoreActionReplayHeader) != "true" {
		t.Fatal("neither response was marked as an idempotent replay")
	}
}

func TestAbyssCoreActionGuardRejectsKeyReuseWithDifferentPayload(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := &WebServer{}
	handler := server.guardAbyssCoreAction(func(w http.ResponseWriter, _ *http.Request, _ string) {
		calls.Add(1)
		writeJSON(w, map[string]any{"ok": true})
	})

	first := httptest.NewRecorder()
	handler(first, newAbyssCoreActionRequest("stable-key-123", []byte(`{"index":1}`)), "player-1")
	second := httptest.NewRecorder()
	handler(second, newAbyssCoreActionRequest("stable-key-123", []byte(`{"index":2}`)), "player-1")

	if second.Code != http.StatusConflict {
		t.Fatalf("second status = %d, want %d", second.Code, http.StatusConflict)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("handler calls = %d, want 1", got)
	}
}

func TestAbyssCoreActionGuardRateLimitAndRefill(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC)
	server := &WebServer{}
	server.abyssCoreActions.now = func() time.Time { return now }
	var calls atomic.Int32
	handler := server.guardAbyssCoreAction(func(w http.ResponseWriter, _ *http.Request, _ string) {
		calls.Add(1)
		writeJSON(w, map[string]any{"ok": true})
	})

	for i := 0; i < abyssCoreActionRateBurst; i++ {
		response := httptest.NewRecorder()
		handler(response, newAbyssCoreActionRequest("", nil), "player-1")
		if response.Code != http.StatusOK {
			t.Fatalf("burst request %d status = %d, want 200", i+1, response.Code)
		}
	}
	limited := httptest.NewRecorder()
	handler(limited, newAbyssCoreActionRequest("", nil), "player-1")
	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("limited status = %d, want %d", limited.Code, http.StatusTooManyRequests)
	}
	if limited.Header().Get("Retry-After") == "" {
		t.Fatal("limited response has no Retry-After header")
	}
	var payload struct {
		RetryAfterMS int `json:"retry_after_ms"`
	}
	if err := json.Unmarshal(limited.Body.Bytes(), &payload); err != nil || payload.RetryAfterMS <= 0 {
		t.Fatalf("limited response retry payload = %+v, error = %v", payload, err)
	}

	now = now.Add(500 * time.Millisecond)
	refilled := httptest.NewRecorder()
	handler(refilled, newAbyssCoreActionRequest("", nil), "player-1")
	if refilled.Code != http.StatusOK {
		t.Fatalf("refilled status = %d, want 200", refilled.Code)
	}
	if got := calls.Load(); got != abyssCoreActionRateBurst+1 {
		t.Fatalf("handler calls = %d, want %d", got, abyssCoreActionRateBurst+1)
	}
}

func TestAbyssCoreActionGuardValidatesKey(t *testing.T) {
	t.Parallel()

	server := &WebServer{}
	var calls atomic.Int32
	handler := server.guardAbyssCoreAction(func(w http.ResponseWriter, _ *http.Request, _ string) {
		calls.Add(1)
		writeJSON(w, map[string]any{"ok": true})
	})

	for _, key := range []string{"short", "invalid key", string(bytes.Repeat([]byte("x"), 129))} {
		response := httptest.NewRecorder()
		handler(response, newAbyssCoreActionRequest(key, nil), "player-1")
		if response.Code != http.StatusBadRequest {
			t.Errorf("key %q status = %d, want %d", key, response.Code, http.StatusBadRequest)
		}
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("handler calls = %d, want 0", got)
	}
}

func TestAbyssCoreActionGuardDetectsRepeatedThrottledBurst(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC)
	guard := abyssCoreActionGuard{now: func() time.Time { return now }}
	for i := 0; i < abyssCoreActionRateBurst; i++ {
		if retryAfter, macro := guard.takeRateToken("player-1"); retryAfter != 0 || macro {
			t.Fatalf("burst token %d = (%s, %t), want (0, false)", i+1, retryAfter, macro)
		}
	}
	for denied := 1; denied <= 3; denied++ {
		retryAfter, macro := guard.takeRateToken("player-1")
		if retryAfter <= 0 {
			t.Fatalf("denial %d retry = %s, want positive duration", denied, retryAfter)
		}
		if macro != (denied == 3) {
			t.Fatalf("denial %d macro = %t, want %t", denied, macro, denied == 3)
		}
	}
}

func TestAbyssDescendGuardPeakConcurrency(t *testing.T) {
	t.Parallel()

	const clients = 128
	server := &WebServer{}
	entered := make(chan struct{}, clients)
	release := make(chan struct{})
	handler := server.guardAbyssCoreAction(func(w http.ResponseWriter, _ *http.Request, uid string) {
		unlock := server.lockAbyss(uid)
		defer unlock()
		entered <- struct{}{}
		<-release
		writeJSON(w, map[string]any{"ok": true})
	})

	var wait sync.WaitGroup
	errors := make(chan string, clients)
	for index := 0; index < clients; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			request := newAbyssCoreActionRequest(fmt.Sprintf("load-key-%04d", index), []byte(`{"interactive":true}`))
			response := httptest.NewRecorder()
			handler(response, request, fmt.Sprintf("player-%04d", index))
			if response.Code != http.StatusOK {
				errors <- fmt.Sprintf("client %d status = %d", index, response.Code)
			}
		}(index)
	}

	deadline := time.After(10 * time.Second)
	for count := 0; count < clients; count++ {
		select {
		case <-entered:
		case <-deadline:
			close(release)
			t.Fatalf("only %d/%d concurrent descend requests reached the handler", count, clients)
		}
	}
	close(release)
	wait.Wait()
	close(errors)
	for message := range errors {
		t.Error(message)
	}
}

func newAbyssCoreActionRequest(key string, body []byte) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/descend", bytes.NewReader(body))
	if key != "" {
		request.Header.Set(abyssCoreActionRequestIDHeader, key)
	}
	return request
}
