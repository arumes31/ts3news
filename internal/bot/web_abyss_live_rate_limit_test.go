package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandleAbyssCombatActionReturnsExhaustedBudgetState(t *testing.T) {
	idempotency := make(map[string]abyssLiveIdempotency, abyssLiveMaxIdempotencyKeysPerRound)
	for index := range abyssLiveMaxIdempotencyKeysPerRound {
		idempotency[fmt.Sprintf("user:key-%d", index)] = abyssLiveIdempotency{Round: 4}
	}
	combat := &abyssLiveCombat{
		id:           "session",
		ownerUID:     "user",
		participants: map[string]bool{"user": true},
		tactics:      map[string]string{"user": "balanced"},
		policies:     map[string]abyssLivePolicy{},
		phase:        "planning",
		round:        4,
		deadline:     time.Now().Add(time.Minute),
		options:      map[string][]abyssLiveOption{},
		queued:       map[string]abyssLiveAction{},
		ready:        map[string]bool{},
		timeBank:     map[string]time.Duration{},
		idempotency:  idempotency,
	}
	server := &WebServer{}
	server.liveCombats.Store("session", combat)
	server.liveCombatByUID.Store("user", "session")

	request := httptest.NewRequest(http.MethodPost, "/api/abyss/combat/action", bytes.NewBufferString(
		`{"session_id":"session","kind":"attack","target_id":"enemy:0","round":4,"idempotency_key":"overflow"}`,
	))
	response := httptest.NewRecorder()
	server.handleAbyssCombatAction(response, request, "user")

	var payload struct {
		OK    bool              `json:"ok"`
		Error string            `json:"error"`
		State abyssLiveSnapshot `json:"state"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.OK || !strings.Contains(payload.Error, "queued action remains active") {
		t.Fatalf("unexpected limit response: %+v", payload)
	}
	if payload.State.ActionBudget.Remaining != 0 ||
		payload.State.ActionBudget.Limit != abyssLiveMaxIdempotencyKeysPerRound {
		t.Fatalf(
			"response action budget = %d/%d, want 0/%d",
			payload.State.ActionBudget.Remaining,
			payload.State.ActionBudget.Limit,
			abyssLiveMaxIdempotencyKeysPerRound,
		)
	}
}
