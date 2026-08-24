package bot

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"ts3news/internal/content"
)

const (
	abyssTreeIdempotencyHeader = "Idempotency-Key"
	abyssTreeRenderMSHeader    = "X-Abyss-Tree-Render-Ms"
	abyssTreeRequestMaxBytes   = 256 * 1024
)

type abyssTreeMutationResponse struct {
	OK            bool   `json:"ok"`
	Error         string `json:"error"`
	Message       string `json:"msg"`
	Mutation      string `json:"mutation"`
	SchemaVersion int    `json:"schema_version"`
	LayoutHash    string `json:"layout_hash"`
}

type abyssTreeOpsMetrics struct {
	previews      atomic.Int64
	commits       atomic.Int64
	refunds       atomic.Int64
	respecs       atomic.Int64
	failures      atomic.Int64
	mutationCount atomic.Int64
	mutationNanos atomic.Int64
	renderCount   atomic.Int64
	renderNanos   atomic.Int64

	mu          sync.Mutex
	sectors     map[int]int64
	nodeKinds   map[string]int64
	idempotency sync.Map
	locks       sync.Map
}

type abyssTreeCachedResponse struct {
	RequestHash [sha256.Size]byte
	Status      int
	Header      http.Header
	Body        []byte
	ExpiresAt   time.Time
}

type abyssTreeBufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newAbyssTreeBufferedResponse() *abyssTreeBufferedResponse {
	return &abyssTreeBufferedResponse{header: make(http.Header)}
}

func (w *abyssTreeBufferedResponse) Header() http.Header { return w.header }

func (w *abyssTreeBufferedResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *abyssTreeBufferedResponse) Write(payload []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(payload)
}

func (w *abyssTreeBufferedResponse) successfulJSON() bool {
	var result struct {
		OK bool `json:"ok"`
	}
	return w.status >= 200 && w.status < 300 && json.Unmarshal(w.body.Bytes(), &result) == nil && result.OK
}

func writeAbyssTreeBufferedResponse(w http.ResponseWriter, response *abyssTreeBufferedResponse) {
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

func (s *WebServer) requireAbyssTreeEnhancements(next abyssTreeHandler) abyssTreeHandler {
	return func(w http.ResponseWriter, r *http.Request, uid string) {
		if !s.abyssFeatures.enabled("tree", uid) {
			writeJSON(w, map[string]any{"ok": false, "error": "enhanced skill-tree features are temporarily disabled"})
			return
		}
		next(w, r, uid)
	}
}

func (s *WebServer) abyssTreeMutation(kind string, next abyssTreeHandler) abyssTreeHandler {
	return s.requireAbyssTreeLayout(s.idempotentAbyssTreeMutation(s.observeAbyssTreeRequest(kind, next)))
}

func (s *WebServer) abyssTreeEnhancedMutation(kind string, next abyssTreeHandler) abyssTreeHandler {
	return s.requireAbyssTreeEnhancements(s.abyssTreeMutation(kind, next))
}

func (s *WebServer) abyssTreePreview(next abyssTreeHandler) abyssTreeHandler {
	return s.requireAbyssTreeEnhancements(s.requireAbyssTreeLayout(s.observeAbyssTreeRequest("preview", next)))
}

func (s *WebServer) observeAbyssTreeRequest(kind string, next abyssTreeHandler) abyssTreeHandler {
	return func(w http.ResponseWriter, r *http.Request, uid string) {
		started := time.Now()
		if err := r.Context().Err(); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "request cancelled"})
			return
		}
		payload, err := readBoundedAbyssTreeBody(r.Body)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(payload))
		response := newAbyssTreeBufferedResponse()
		next(response, r, uid)
		normalizeAbyssTreeMutationResponse(response, kind)
		s.abyssTreeOps.observe(kind, time.Since(started), r.Header.Get(abyssTreeRenderMSHeader), payload, response.successfulJSON())
		s.recordAbyssTreeAudit(r, uid, kind, payload, response.successfulJSON())
		writeAbyssTreeBufferedResponse(w, response)
	}
}

func (s *WebServer) idempotentAbyssTreeMutation(next abyssTreeHandler) abyssTreeHandler {
	return func(w http.ResponseWriter, r *http.Request, uid string) {
		key := r.Header.Get(abyssTreeIdempotencyHeader)
		if key == "" {
			next(w, r, uid)
			return
		}
		if len(key) > 128 {
			writeJSON(w, map[string]any{"ok": false, "error": "idempotency key is too long"})
			return
		}
		payload, err := readBoundedAbyssTreeBody(r.Body)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(payload))
		requestHash := sha256.Sum256(payload)
		cacheKey := uid + "\x00" + r.URL.Path + "\x00" + key
		lockValue, _ := s.abyssTreeOps.locks.LoadOrStore(cacheKey, &sync.Mutex{})
		lock := lockValue.(*sync.Mutex)
		lock.Lock()
		defer lock.Unlock()

		if value, ok := s.abyssTreeOps.idempotency.Load(cacheKey); ok {
			cached := value.(abyssTreeCachedResponse)
			if time.Now().Before(cached.ExpiresAt) {
				if cached.RequestHash != requestHash {
					writeJSON(w, map[string]any{"ok": false, "error": "idempotency key was already used with another payload"})
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
			s.abyssTreeOps.idempotency.Delete(cacheKey)
		}

		response := newAbyssTreeBufferedResponse()
		next(response, r, uid)
		if response.successfulJSON() {
			s.abyssTreeOps.idempotency.Store(cacheKey, abyssTreeCachedResponse{
				RequestHash: requestHash, Status: max(response.status, http.StatusOK),
				Header: response.header.Clone(), Body: append([]byte(nil), response.body.Bytes()...),
				ExpiresAt: time.Now().Add(15 * time.Minute),
			})
		}
		writeAbyssTreeBufferedResponse(w, response)
	}
}

func readBoundedAbyssTreeBody(body io.Reader) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	payload, err := io.ReadAll(io.LimitReader(body, abyssTreeRequestMaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(payload) > abyssTreeRequestMaxBytes {
		return nil, fmt.Errorf("tree request exceeds %d bytes", abyssTreeRequestMaxBytes)
	}
	return payload, nil
}

func normalizeAbyssTreeMutationResponse(response *abyssTreeBufferedResponse, kind string) {
	var payload map[string]any
	if json.Unmarshal(response.body.Bytes(), &payload) != nil {
		return
	}
	if _, ok := payload["ok"]; !ok {
		payload["ok"] = false
	}
	if _, ok := payload["error"]; !ok {
		payload["error"] = ""
	}
	if _, ok := payload["msg"]; !ok {
		payload["msg"] = ""
	}
	payload["mutation"] = kind
	payload["schema_version"] = content.TreeCatalogSchemaVersion
	payload["layout_hash"] = content.AbyssTree().TopologyHash()
	encoded, err := json.Marshal(payload)
	if err == nil {
		response.body.Reset()
		_, _ = response.body.Write(encoded)
	}
}

func (s *WebServer) recordAbyssTreeAudit(r *http.Request, uid, kind string, payload []byte, success bool) {
	if s.bot == nil || s.bot.DB == nil {
		return
	}
	var request struct {
		NodeID int   `json:"node_id"`
		FromID int   `json:"from_id"`
		ToID   int   `json:"to_id"`
		IDs    []int `json:"ids"`
	}
	_ = json.Unmarshal(payload, &request)
	ids := append([]int(nil), request.IDs...)
	ids = append(ids, request.NodeID, request.FromID, request.ToID)
	canonical := make([]int, 0, len(ids))
	seen := map[int]bool{}
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			canonical = append(canonical, id)
		}
	}
	encodedIDs, _ := json.Marshal(canonical)
	if _, err := s.bot.DB.ExecContext(
		r.Context(),
		`INSERT INTO abyss_tree_mutation_audit (client_uid, action, node_ids, succeeded, request_key)
		 VALUES ($1,$2,$3::jsonb,$4,$5)`,
		uid,
		kind,
		string(encodedIDs),
		success,
		r.Header.Get(abyssTreeIdempotencyHeader),
	); err != nil {
		return
	}
	_, _ = s.bot.DB.ExecContext(
		r.Context(),
		`DELETE FROM abyss_tree_mutation_audit
		 WHERE client_uid=$1 AND id NOT IN
		       (SELECT id FROM abyss_tree_mutation_audit WHERE client_uid=$1 ORDER BY id DESC LIMIT 50)`,
		uid,
	)
}

func (m *abyssTreeOpsMetrics) observe(kind string, elapsed time.Duration, renderHeader string, payload []byte, success bool) {
	switch kind {
	case "preview":
		m.previews.Add(1)
	case "refund":
		m.refunds.Add(1)
	case "respec":
		m.respecs.Add(1)
	default:
		m.commits.Add(1)
	}
	if !success {
		m.failures.Add(1)
	}
	if kind != "preview" {
		m.mutationCount.Add(1)
		m.mutationNanos.Add(elapsed.Nanoseconds())
	}
	if renderMS, err := strconv.ParseFloat(renderHeader, 64); err == nil && renderMS >= 0 && renderMS <= 60000 {
		m.renderCount.Add(1)
		m.renderNanos.Add(int64(renderMS * float64(time.Millisecond)))
	}
	if success {
		m.observeAnonymousNodes(payload)
	}
}

func (m *abyssTreeOpsMetrics) observeAnonymousNodes(payload []byte) {
	var request struct {
		NodeID int   `json:"node_id"`
		FromID int   `json:"from_id"`
		ToID   int   `json:"to_id"`
		IDs    []int `json:"ids"`
	}
	if json.Unmarshal(payload, &request) != nil {
		return
	}
	ids := append([]int(nil), request.IDs...)
	ids = append(ids, request.NodeID, request.FromID, request.ToID)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sectors == nil {
		m.sectors = make(map[int]int64)
		m.nodeKinds = make(map[string]int64)
	}
	seen := make(map[int]bool, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		if node := content.AbyssTree().Node(id); node != nil {
			m.sectors[node.Sector]++
			m.nodeKinds[node.Type]++
		}
	}
}

func (m *abyssTreeOpsMetrics) snapshot() map[string]any {
	m.mu.Lock()
	sectors := make(map[int]int64, len(m.sectors))
	kinds := make(map[string]int64, len(m.nodeKinds))
	for key, value := range m.sectors {
		sectors[key] = value
	}
	for key, value := range m.nodeKinds {
		kinds[key] = value
	}
	m.mu.Unlock()
	mutations := m.mutationCount.Load()
	renders := m.renderCount.Load()
	return map[string]any{
		"counts": map[string]int64{
			"preview": m.previews.Load(), "commit": m.commits.Load(), "refund": m.refunds.Load(),
			"respec": m.respecs.Load(), "failure": m.failures.Load(),
		},
		"latency_ms": map[string]int64{
			"mutation_avg": durationAverage(m.mutationNanos.Load(), mutations),
			"render_avg":   durationAverage(m.renderNanos.Load(), renders),
		},
		"anonymous_popularity": map[string]any{"sectors": sectors, "node_kinds": kinds},
	}
}
