package bot

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

const abyssFunnelMaxActive = 10_000

type abyssFunnelRun struct {
	startedAt time.Time
	reached5  bool
}

// abyssFunnelMetrics tracks process-lifetime transitions without exposing or
// retaining raw player IDs. The bounded active map makes floor and terminal
// transitions idempotent while keeping the operator output aggregate-only.
type abyssFunnelMetrics struct {
	mu        sync.Mutex
	active    map[string]abyssFunnelRun
	maxActive int
	entered   int64
	reached5  int64
	banked    int64
	banked5   int64
	conceded  int64
	evicted   int64
}

func abyssAnonymousPlayerRef(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func (m *abyssFunnelMetrics) observeEnter(uid string, now time.Time) {
	ref := abyssAnonymousPlayerRef(uid)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active == nil {
		m.active = make(map[string]abyssFunnelRun)
	}
	if _, exists := m.active[ref]; !exists && len(m.active) >= m.activeLimit() {
		m.evictOldestLocked()
	}
	m.active[ref] = abyssFunnelRun{startedAt: now.UTC()}
	m.entered++
}

func (m *abyssFunnelMetrics) observeFloor(uid string, depth int) {
	if depth < 5 {
		return
	}
	ref := abyssAnonymousPlayerRef(uid)
	m.mu.Lock()
	defer m.mu.Unlock()
	run, exists := m.active[ref]
	if !exists || run.reached5 {
		return
	}
	run.reached5 = true
	m.active[ref] = run
	m.reached5++
}

func (m *abyssFunnelMetrics) observeBank(uid string) {
	ref := abyssAnonymousPlayerRef(uid)
	m.mu.Lock()
	defer m.mu.Unlock()
	run, exists := m.active[ref]
	if !exists {
		return
	}
	m.banked++
	if run.reached5 {
		m.banked5++
	}
	delete(m.active, ref)
}

func (m *abyssFunnelMetrics) observeConcede(uid string) {
	ref := abyssAnonymousPlayerRef(uid)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.active[ref]; !exists {
		return
	}
	m.conceded++
	delete(m.active, ref)
}

func (m *abyssFunnelMetrics) snapshot() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]any{
		"scope":                  "process_lifetime",
		"entered":                m.entered,
		"reached_floor_5":        m.reached5,
		"banked":                 m.banked,
		"banked_after_floor_5":   m.banked5,
		"conceded":               m.conceded,
		"active_tracked":         len(m.active),
		"evicted":                m.evicted,
		"floor_5_rate":           ratio(m.reached5, m.entered),
		"bank_rate":              ratio(m.banked, m.entered),
		"post_floor_5_bank_rate": ratio(m.banked5, m.reached5),
	}
}

func (m *abyssFunnelMetrics) activeLimit() int {
	if m.maxActive > 0 {
		return m.maxActive
	}
	return abyssFunnelMaxActive
}

func (m *abyssFunnelMetrics) evictOldestLocked() {
	oldestRef := ""
	var oldest time.Time
	for ref, run := range m.active {
		if oldestRef == "" || run.startedAt.Before(oldest) {
			oldestRef = ref
			oldest = run.startedAt
		}
	}
	if oldestRef != "" {
		delete(m.active, oldestRef)
		m.evicted++
	}
}
