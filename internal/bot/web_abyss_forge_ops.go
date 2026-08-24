package bot

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ts3news/internal/content"
)

type abyssForgeResponseWriter struct {
	*abyssTreeBufferedResponse
	ctx context.Context
}

func (w *abyssForgeResponseWriter) forgeContext() context.Context { return w.ctx }

func forgeContextFromWriter(w http.ResponseWriter) context.Context {
	if contextual, ok := w.(interface{ forgeContext() context.Context }); ok {
		return contextual.forgeContext()
	}
	return context.Background()
}

func (s *WebServer) beginForgeRequestTx(w http.ResponseWriter) (*sql.Tx, error) {
	return s.bot.DB.BeginTx(forgeContextFromWriter(w), nil)
}

type abyssForgeOpsMetrics struct {
	quotes        atomic.Int64
	commits       atomic.Int64
	successes     atomic.Int64
	failures      atomic.Int64
	cancellations atomic.Int64
	abandonments  atomic.Int64
	anomalies     atomic.Int64
	quoteNanos    atomic.Int64
	quoteToCommit atomic.Int64
	quoteCommits  atomic.Int64
	commitNanos   atomic.Int64
	idempotency   sync.Map
	locks         sync.Map
	pendingQuotes sync.Map
	operations    sync.Map
}

type abyssForgeOperationMetrics struct {
	commits       atomic.Int64
	successes     atomic.Int64
	failures      atomic.Int64
	cancellations atomic.Int64
}

type abyssForgePendingQuote struct {
	operation string
	createdAt time.Time
	expiresAt time.Time
}

func (m *abyssForgeOpsMetrics) snapshot() map[string]any {
	m.reapAbandonedQuotes(time.Now())
	operations := map[string]map[string]int64{}
	m.operations.Range(func(key, value any) bool {
		metrics := value.(*abyssForgeOperationMetrics)
		operations[key.(string)] = map[string]int64{
			"commits": metrics.commits.Load(), "successes": metrics.successes.Load(),
			"failures": metrics.failures.Load(), "cancellations": metrics.cancellations.Load(),
		}
		return true
	})
	return map[string]any{
		"quotes": m.quotes.Load(), "commits": m.commits.Load(), "successes": m.successes.Load(),
		"failures": m.failures.Load(), "cancellations": m.cancellations.Load(),
		"abandonments": m.abandonments.Load(), "anomalies": m.anomalies.Load(),
		"quote_latency_ms":   durationAverage(m.quoteNanos.Load(), m.quotes.Load()),
		"quote_to_commit_ms": durationAverage(m.quoteToCommit.Load(), m.quoteCommits.Load()),
		"commit_latency_ms":  durationAverage(m.commitNanos.Load(), m.commits.Load()),
		"operations":         operations,
	}
}

func (m *abyssForgeOpsMetrics) operation(id string) *abyssForgeOperationMetrics {
	value, _ := m.operations.LoadOrStore(id, &abyssForgeOperationMetrics{})
	return value.(*abyssForgeOperationMetrics)
}

func (m *abyssForgeOpsMetrics) trackQuote(token, operation string, createdAt, expiresAt time.Time) {
	m.reapAbandonedQuotes(time.Now())
	m.pendingQuotes.Store(token, abyssForgePendingQuote{operation: operation, createdAt: createdAt, expiresAt: expiresAt})
}

func (m *abyssForgeOpsMetrics) consumeQuote(token string) {
	m.reapAbandonedQuotes(time.Now())
	if value, loaded := m.pendingQuotes.LoadAndDelete(token); loaded {
		pending := value.(abyssForgePendingQuote)
		m.quoteToCommit.Add(max(0, time.Since(pending.createdAt).Nanoseconds()))
		m.quoteCommits.Add(1)
	}
}

func (m *abyssForgeOpsMetrics) reapAbandonedQuotes(now time.Time) {
	m.pendingQuotes.Range(func(key, value any) bool {
		if !now.Before(value.(abyssForgePendingQuote).expiresAt) {
			if _, loaded := m.pendingQuotes.LoadAndDelete(key); loaded {
				m.abandonments.Add(1)
			}
		}
		return true
	})
}

func forgeRequestIdentity(payload []byte, operation string) (int64, string, string, json.RawMessage) {
	var values map[string]json.RawMessage
	if json.Unmarshal(payload, &values) != nil {
		return 0, "", "", json.RawMessage(`{}`)
	}
	var invID int64
	var slot, confirmation string
	_ = json.Unmarshal(values["inv_id"], &invID)
	_ = json.Unmarshal(values["slot"], &slot)
	_ = json.Unmarshal(values["confirmation"], &confirmation)
	delete(values, "inv_id")
	if operation != "target_craft" {
		delete(values, "slot")
		slot = strings.TrimSpace(slot)
	} else {
		slot = ""
	}
	delete(values, "confirmation")
	parameters, _ := json.Marshal(values)
	canonical, _ := canonicalForgeParameters(parameters)
	return invID, slot, confirmation, canonical
}

func forgeHandlerPayload(payload []byte) []byte {
	var values map[string]json.RawMessage
	if json.Unmarshal(payload, &values) != nil {
		return payload
	}
	delete(values, "confirmation")
	clean, err := json.Marshal(values)
	if err != nil {
		return payload
	}
	return clean
}

func (s *WebServer) validateForgeCommitQuote(r *http.Request, uid, operation string, payload []byte) error {
	token := r.Header.Get(abyssForgeQuoteHeader)
	if token == "" {
		if s.bot == nil || !s.abyssFeatures.enabled("forge", uid) {
			return nil
		}
		return errForgeQuoteRequired
	}
	claims, err := s.verifyForgeClaims(token)
	if err != nil {
		return err
	}
	invID, slot, confirmation, parameters := forgeRequestIdentity(payload, operation)
	if claims.UID != uid || claims.Operation != operation || claims.InvID != invID || claims.Slot != slot ||
		!bytes.Equal(claims.Parameters, parameters) {
		return errForgeQuoteAltered
	}
	currentRevision, err := s.forgeInventoryRevision(r.Context(), uid)
	if err != nil || currentRevision != claims.Inventory {
		return errForgeQuoteStaleInventory
	}
	if claims.Gear != "none" {
		gear, raw, ok := s.loadForgeQuoteItem(r.Context(), uid, invID, slot)
		if !ok || forgeGearFingerprint(gear, raw, invID, slot) != claims.Gear {
			return errForgeQuoteStaleGear
		}
	}
	if catalog, ok := abyssForgeOperationByID(operation); ok && !catalog.Reversible &&
		confirmation != "FORGE "+stringsToUpper(operation) {
		return errForgeConfirmation
	}
	return nil
}

var (
	errForgeQuoteAltered        = forgeContractError("forge quote parameters were altered")
	errForgeQuoteRequired       = forgeContractError("a fresh server-authored forge quote is required")
	errForgeQuoteStaleInventory = forgeContractError("inventory changed; refresh the forge quote")
	errForgeQuoteStaleGear      = forgeContractError("item changed; refresh the forge quote")
	errForgeConfirmation        = forgeContractError("irreversible operation confirmation is missing")
)

type forgeContractError string

func (err forgeContractError) Error() string { return string(err) }

func stringsToUpper(value string) string {
	result := []byte(value)
	for i, char := range result {
		if char >= 'a' && char <= 'z' {
			result[i] = char - ('a' - 'A')
		}
	}
	return string(result)
}

func (s *WebServer) forgeMutation(operation string, next abyssTreeHandler) abyssTreeHandler {
	return func(w http.ResponseWriter, r *http.Request, uid string) {
		started := time.Now()
		operationMetrics := s.abyssForgeOps.operation(operation)
		if err := r.Context().Err(); err != nil {
			s.abyssForgeOps.cancellations.Add(1)
			operationMetrics.cancellations.Add(1)
			writeJSON(w, map[string]any{"ok": false, "error": "request cancelled"})
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, abyssForgeRequestMax+1))
		if err != nil || len(payload) > abyssForgeRequestMax {
			s.abyssForgeOps.failures.Add(1)
			operationMetrics.failures.Add(1)
			writeJSON(w, map[string]any{"ok": false, "error": "forge request is too large"})
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(payload))
		key := r.Header.Get(abyssForgeIdempotencyKey)
		requestHash := sha256.Sum256(payload)
		cacheKey := uid + "\x00" + operation + "\x00" + key
		var lock *sync.Mutex
		if key != "" {
			if len(key) > 128 {
				s.abyssForgeOps.failures.Add(1)
				operationMetrics.failures.Add(1)
				writeJSON(w, map[string]any{"ok": false, "error": "idempotency key is too long"})
				return
			}
			value, _ := s.abyssForgeOps.locks.LoadOrStore(cacheKey, &sync.Mutex{})
			lock = value.(*sync.Mutex)
			lock.Lock()
			defer lock.Unlock()
			if cachedValue, ok := s.abyssForgeOps.idempotency.Load(cacheKey); ok {
				cached := cachedValue.(abyssTreeCachedResponse)
				if time.Now().Before(cached.ExpiresAt) {
					if cached.RequestHash != requestHash {
						s.abyssForgeOps.failures.Add(1)
						operationMetrics.failures.Add(1)
						writeJSON(w, map[string]any{"ok": false, "error": "idempotency key already used with another request"})
						return
					}
					for header, values := range cached.Header {
						w.Header()[header] = append([]string(nil), values...)
					}
					w.Header().Set("X-Idempotent-Replay", "true")
					w.WriteHeader(cached.Status)
					_, _ = w.Write(cached.Body)
					return
				}
			}
		}
		if err := s.validateForgeCommitQuote(r, uid, operation, payload); err != nil {
			s.abyssForgeOps.failures.Add(1)
			operationMetrics.failures.Add(1)
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		s.abyssForgeOps.consumeQuote(r.Header.Get(abyssForgeQuoteHeader))

		before := s.forgeAuditSnapshot(r, uid, operation, payload)
		buffer := newAbyssTreeBufferedResponse()
		response := &abyssForgeResponseWriter{abyssTreeBufferedResponse: buffer, ctx: r.Context()}
		r.Body = io.NopCloser(bytes.NewReader(forgeHandlerPayload(payload)))
		next(response, r, uid)
		success := buffer.successfulJSON()
		s.abyssForgeOps.commits.Add(1)
		operationMetrics.commits.Add(1)
		s.abyssForgeOps.commitNanos.Add(time.Since(started).Nanoseconds())
		if success {
			s.abyssForgeOps.successes.Add(1)
			operationMetrics.successes.Add(1)
		} else {
			s.abyssForgeOps.failures.Add(1)
			operationMetrics.failures.Add(1)
		}
		if r.Context().Err() != nil {
			s.abyssForgeOps.cancellations.Add(1)
			operationMetrics.cancellations.Add(1)
		}
		after := s.forgeAuditSnapshot(r, uid, operation, payload)
		if forgeMutationAnomalous(before, after) {
			s.abyssForgeOps.anomalies.Add(1)
			log.Printf("forge anomaly detected for operation %s", operation)
		}
		s.recordForgeMutationAudit(r, uid, operation, before, after, success)
		if success && key != "" {
			s.abyssForgeOps.idempotency.Store(cacheKey, abyssTreeCachedResponse{
				RequestHash: requestHash, Status: max(buffer.status, http.StatusOK), Header: buffer.header.Clone(),
				Body: append([]byte(nil), buffer.body.Bytes()...), ExpiresAt: time.Now().Add(15 * time.Minute),
			})
		}
		writeAbyssTreeBufferedResponse(w, buffer)
	}
}

func forgeMutationAnomalous(before, after json.RawMessage) bool {
	var oldState, newState struct {
		Item      content.Gear     `json:"item"`
		Materials map[string]int64 `json:"materials"`
	}
	if json.Unmarshal(before, &oldState) != nil || json.Unmarshal(after, &newState) != nil {
		return false
	}
	oldCR, newCR := oldState.Item.CombatRating(), newState.Item.CombatRating()
	if oldCR > 0 && newCR > oldCR*4+100 {
		return true
	}
	for material, value := range newState.Materials {
		if value-oldState.Materials[material] > 10000 {
			return true
		}
	}
	return false
}

func (s *WebServer) forgeAuditSnapshot(r *http.Request, uid, operation string, payload []byte) json.RawMessage {
	if s.bot == nil || s.bot.DB == nil {
		return json.RawMessage(`null`)
	}
	invID, slot, _, _ := forgeRequestIdentity(payload, operation)
	gear, raw, ok := s.loadForgeQuoteItem(r.Context(), uid, invID, slot)
	snapshot := map[string]any{
		"gold": s.bot.abyssGold(uid), "tokens": s.bot.abyssTokens(uid), "materials": s.bot.loadMaterials(uid),
	}
	if ok {
		snapshot["fingerprint"] = forgeGearFingerprint(gear, raw, invID, slot)
		snapshot["item"] = gear
	}
	encoded, _ := json.Marshal(snapshot)
	return encoded
}

func (s *WebServer) recordForgeMutationAudit(r *http.Request, uid, operation string, before, after json.RawMessage, success bool) {
	if s.bot == nil || s.bot.DB == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 2*time.Second)
	defer cancel()
	if _, err := s.bot.DB.ExecContext(ctx,
		`INSERT INTO abyss_forge_mutation_audit (client_uid, operation, before_state, after_state, succeeded, request_key)
		 VALUES ($1,$2,$3::jsonb,$4::jsonb,$5,$6)`, uid, operation, string(before), string(after), success,
		r.Header.Get(abyssForgeIdempotencyKey)); err != nil {
		log.Printf("forge audit insert failed for operation %s: %v", operation, err)
		return
	}
	_, _ = s.bot.DB.ExecContext(ctx,
		`DELETE FROM abyss_forge_mutation_audit WHERE client_uid=$1 AND id NOT IN
		 (SELECT id FROM abyss_forge_mutation_audit WHERE client_uid=$1 ORDER BY id DESC LIMIT 100)`, uid)
}

func parseForgeRenderMillis(value string) int64 {
	millis, err := strconv.ParseInt(value, 10, 64)
	if err != nil || millis < 0 || millis > 60000 {
		return 0
	}
	return millis
}
