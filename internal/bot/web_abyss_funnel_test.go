package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAbyssFunnelMetricsDeduplicateAndCloseRuns(t *testing.T) {
	t.Parallel()

	var metrics abyssFunnelMetrics
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	metrics.observeEnter("player-1", now)
	metrics.observeFloor("player-1", 4)
	metrics.observeFloor("player-1", 5)
	metrics.observeFloor("player-1", 12)
	metrics.observeBank("player-1")
	metrics.observeBank("player-1")

	snapshot := metrics.snapshot()
	for key, want := range map[string]int64{
		"entered": 1, "reached_floor_5": 1, "banked": 1,
		"banked_after_floor_5": 1, "conceded": 0,
	} {
		if got := snapshot[key]; got != want {
			t.Errorf("%s = %v, want %d", key, got, want)
		}
	}
	if got := snapshot["active_tracked"]; got != 0 {
		t.Errorf("active_tracked = %v, want 0", got)
	}
}

func TestAbyssFunnelOpsSnapshotContainsNoPlayerIdentifiers(t *testing.T) {
	t.Parallel()

	const privateUID = "private-player-identity"
	server := &WebServer{abyssFeatures: abyssFeatureConfig{opsToken: "operator-secret"}}
	server.abyssOps.funnel.observeEnter(privateUID, time.Now())
	server.abyssOps.funnel.observeFloor(privateUID, 5)

	request := httptest.NewRequest(http.MethodGet, "/api/abyss/ops", nil)
	request.Header.Set("Authorization", "Bearer operator-secret")
	recorder := httptest.NewRecorder()
	server.handleAbyssOps(recorder, request, "operator")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), privateUID) ||
		strings.Contains(recorder.Body.String(), abyssAnonymousPlayerRef(privateUID)) {
		t.Fatal("operator funnel response leaked a raw or pseudonymous player identifier")
	}
	var response map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode operator response: %v", err)
	}
	var funnel map[string]any
	if err := json.Unmarshal(response["funnel"], &funnel); err != nil {
		t.Fatalf("decode funnel response: %v", err)
	}
	if funnel["entered"] != float64(1) || funnel["reached_floor_5"] != float64(1) {
		t.Fatalf("funnel response = %#v", funnel)
	}
}

func TestAbyssFunnelMetricsRemainBounded(t *testing.T) {
	t.Parallel()

	metrics := abyssFunnelMetrics{maxActive: 3}
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	for index := range 5 {
		metrics.observeEnter(fmt.Sprintf("player-%d", index), now.Add(time.Duration(index)*time.Second))
	}
	snapshot := metrics.snapshot()
	if got := snapshot["active_tracked"]; got != 3 {
		t.Fatalf("active_tracked = %v, want 3", got)
	}
	if got := snapshot["evicted"]; got != int64(2) {
		t.Fatalf("evicted = %v, want 2", got)
	}
}

func TestAbyssFunnelMetricsAreConcurrencySafe(t *testing.T) {
	t.Parallel()

	const delvers = 100
	var metrics abyssFunnelMetrics
	var wait sync.WaitGroup
	for index := range delvers {
		wait.Go(func() {
			uid := fmt.Sprintf("delver-%d", index)
			metrics.observeEnter(uid, time.Now())
			metrics.observeFloor(uid, 5)
			metrics.observeBank(uid)
		})
	}
	wait.Wait()
	snapshot := metrics.snapshot()
	for _, key := range []string{"entered", "reached_floor_5", "banked", "banked_after_floor_5"} {
		if got := snapshot[key]; got != int64(delvers) {
			t.Errorf("%s = %v, want %d", key, got, delvers)
		}
	}
}
