package bot

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestIdempotentAbyssTreeMutationReplaysSuccessfulCommit(t *testing.T) {
	server := &WebServer{}
	var calls atomic.Int64
	handler := server.idempotentAbyssTreeMutation(func(w http.ResponseWriter, _ *http.Request, _ string) {
		calls.Add(1)
		writeJSON(w, map[string]any{"ok": true, "value": 7})
	})
	request := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/abyss/tree/allocate", strings.NewReader(body))
		req.Header.Set(abyssTreeIdempotencyHeader, "same-key")
		recorder := httptest.NewRecorder()
		handler(recorder, req, "user")
		return recorder
	}
	first := request(`{"node_id":1}`)
	second := request(`{"node_id":1}`)
	if calls.Load() != 1 || first.Body.String() != second.Body.String() {
		t.Fatalf("idempotent calls/body = %d, %q / %q", calls.Load(), first.Body.String(), second.Body.String())
	}
	if second.Header().Get("X-Idempotent-Replay") != "true" {
		t.Fatal("replayed response is missing its marker header")
	}
	conflict := request(`{"node_id":2}`)
	if !strings.Contains(conflict.Body.String(), "another payload") || calls.Load() != 1 {
		t.Fatalf("conflicting replay = %q, calls %d", conflict.Body.String(), calls.Load())
	}
}

func TestAbyssTreeOpsMetricsAreAnonymousAndSeparated(t *testing.T) {
	metrics := &abyssTreeOpsMetrics{}
	metrics.observe("preview", 5*time.Millisecond, "3.5", []byte(`{"node_id":1}`), true)
	metrics.observe("commit", 10*time.Millisecond, "4.5", []byte(`{"node_id":1}`), true)
	metrics.observe("refund", 20*time.Millisecond, "", []byte(`{"node_id":1}`), false)
	snapshot := metrics.snapshot()
	counts := snapshot["counts"].(map[string]int64)
	if counts["preview"] != 1 || counts["commit"] != 1 || counts["refund"] != 1 || counts["failure"] != 1 {
		t.Fatalf("tree metric counts = %+v", counts)
	}
	latency := snapshot["latency_ms"].(map[string]int64)
	if latency["mutation_avg"] != 15 || latency["render_avg"] != 4 {
		t.Fatalf("separate mutation/render latency = %+v", latency)
	}
	popularity := snapshot["anonymous_popularity"].(map[string]any)
	if len(popularity["sectors"].(map[int]int64)) == 0 || len(popularity["node_kinds"].(map[string]int64)) == 0 {
		t.Fatalf("anonymous popularity = %+v", popularity)
	}
	encoded, _ := json.Marshal(snapshot)
	if strings.Contains(strings.ToLower(string(encoded)), "user") {
		t.Fatal("anonymous skill-tree metrics unexpectedly contain identity fields")
	}
}

func TestRequireAbyssTreeEnhancementsKillSwitch(t *testing.T) {
	server := &WebServer{abyssFeatures: &abyssFeatureConfig{tree: false}}
	called := false
	handler := server.requireAbyssTreeEnhancements(func(http.ResponseWriter, *http.Request, string) { called = true })
	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest(http.MethodPost, "/api/abyss/tree/plan_preview", nil), "user")
	if called || !strings.Contains(recorder.Body.String(), "temporarily disabled") {
		t.Fatalf("disabled handler called=%v body=%q", called, recorder.Body.String())
	}
}

type fakeAbyssTreeTransaction struct {
	failAt     int
	execs      int
	commitErr  error
	committed  bool
	rolledBack bool
}

func (tx *fakeAbyssTreeTransaction) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	tx.execs++
	if tx.execs == tx.failAt {
		return nil, errors.New("write failed")
	}
	return nil, nil
}

func (tx *fakeAbyssTreeTransaction) Commit() error {
	if tx.commitErr != nil {
		return tx.commitErr
	}
	tx.committed = true
	return nil
}

func (tx *fakeAbyssTreeTransaction) Rollback() error {
	tx.rolledBack = true
	return nil
}

func TestCommitAbyssTreeReplacementRollsBackEveryFailure(t *testing.T) {
	ids := []int{1, 2, 3}
	for failAt := 1; failAt <= len(ids)+1; failAt++ {
		t.Run(strconv.Itoa(failAt), func(t *testing.T) {
			tx := &fakeAbyssTreeTransaction{failAt: failAt}
			if err := commitAbyssTreeReplacement(context.Background(), tx, "user", ids); err == nil || !tx.rolledBack || tx.committed {
				t.Fatalf("write failure %d: err=%v rollback=%v commit=%v", failAt, err, tx.rolledBack, tx.committed)
			}
		})
	}
	tx := &fakeAbyssTreeTransaction{commitErr: errors.New("commit failed")}
	if err := commitAbyssTreeReplacement(context.Background(), tx, "user", ids); err == nil || !tx.rolledBack {
		t.Fatalf("commit failure: err=%v rollback=%v", err, tx.rolledBack)
	}
}

func TestLockAbyssSerializesConcurrentAllocation(t *testing.T) {
	server := &WebServer{}
	unlock := server.lockAbyss("user")
	acquired := make(chan struct{})
	done := make(chan struct{})
	go func() {
		secondUnlock := server.lockAbyss("user")
		close(acquired)
		secondUnlock()
		close(done)
	}()
	select {
	case <-acquired:
		t.Fatal("second mutation acquired the per-user lock concurrently")
	case <-time.After(25 * time.Millisecond):
	}
	unlock()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("second mutation did not acquire the released lock")
	}
}

func TestNormalizeAbyssTreeMutationResponseAddsContractFields(t *testing.T) {
	response := newAbyssTreeBufferedResponse()
	writeJSON(response, map[string]any{"ok": true, "value": 7})
	normalizeAbyssTreeMutationResponse(response, "commit")
	var envelope abyssTreeMutationResponse
	if err := json.Unmarshal(response.body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.OK || envelope.Error != "" || envelope.Message != "" || envelope.Mutation != "commit" ||
		envelope.SchemaVersion == 0 || envelope.LayoutHash == "" {
		t.Fatalf("mutation envelope = %+v", envelope)
	}
}

func TestReadBoundedAbyssTreeBody(t *testing.T) {
	payload, err := readBoundedAbyssTreeBody(strings.NewReader("ok"))
	if err != nil || string(payload) != "ok" {
		t.Fatalf("bounded body = %q, %v", payload, err)
	}
	tooLarge := io.LimitReader(strings.NewReader(strings.Repeat("x", abyssTreeRequestMaxBytes+1)), abyssTreeRequestMaxBytes+1)
	if _, err := readBoundedAbyssTreeBody(tooLarge); err == nil {
		t.Fatal("oversized tree request was accepted")
	}
}

func TestCanonicalAbyssTreeIDsPreservesFirstOccurrence(t *testing.T) {
	got := canonicalAbyssTreeIDs([]int{3, 1, 3, 2, 1})
	want := []int{3, 1, 2}
	if len(got) != len(want) {
		t.Fatalf("canonical IDs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("canonical IDs = %v, want %v", got, want)
		}
	}
}
