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
	maxDepth  int
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
	stops     map[string]map[string]int64
}

func abyssAnonymousPlayerRef(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func (m *abyssFunnelMetrics) observeEnter(uid string, now time.Time, startDepth int) {
	ref := abyssAnonymousPlayerRef(uid)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active == nil {
		m.active = make(map[string]abyssFunnelRun)
	}
	if _, exists := m.active[ref]; !exists && len(m.active) >= m.activeLimit() {
		m.evictOldestLocked()
	}
	m.active[ref] = abyssFunnelRun{startedAt: now.UTC(), maxDepth: max(0, startDepth)}
	m.entered++
}

func (m *abyssFunnelMetrics) observeFloor(uid string, depth int) {
	ref := abyssAnonymousPlayerRef(uid)
	m.mu.Lock()
	defer m.mu.Unlock()
	run, exists := m.active[ref]
	if !exists {
		return
	}
	run.maxDepth = max(run.maxDepth, depth)
	if depth >= 5 && !run.reached5 {
		run.reached5 = true
		m.reached5++
	}
	m.active[ref] = run
}

func (m *abyssFunnelMetrics) observeBank(uid string) {
	m.observeEnd(uid, "banked")
}

func (m *abyssFunnelMetrics) observeEnd(uid, reason string) {
	ref := abyssAnonymousPlayerRef(uid)
	m.mu.Lock()
	defer m.mu.Unlock()
	run, exists := m.active[ref]
	if !exists {
		return
	}
	switch reason {
	case "banked":
		m.banked++
		if run.reached5 {
			m.banked5++
		}
	case "conceded", "timeout", "revive_failed":
		m.conceded++
	default:
		reason = "other"
	}
	if m.stops == nil {
		m.stops = make(map[string]map[string]int64)
	}
	band := abyssFunnelDepthBand(run.maxDepth)
	if m.stops[band] == nil {
		m.stops[band] = make(map[string]int64)
	}
	m.stops[band][reason]++
	delete(m.active, ref)
}

func abyssFunnelDepthBand(depth int) string {
	switch {
	case depth <= 0:
		return "entry"
	case depth < 5:
		return "1-4"
	case depth < 10:
		return "5-9"
	case depth < 25:
		return "10-24"
	case depth < 50:
		return "25-49"
	case depth < 100:
		return "50-99"
	default:
		return "100+"
	}
}

func (m *abyssFunnelMetrics) snapshot() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	stops := make(map[string]map[string]int64, len(m.stops))
	for band, reasons := range m.stops {
		stops[band] = make(map[string]int64, len(reasons))
		for reason, count := range reasons {
			stops[band][reason] = count
		}
	}
	return map[string]any{
		"scope":                  "process_lifetime",
		"entered":                m.entered,
		"reached_floor_5":        m.reached5,
		"banked":                 m.banked,
		"banked_after_floor_5":   m.banked5,
		"conceded":               m.conceded,
		"active_tracked":         len(m.active),
		"evicted":                m.evicted,
		"stops_by_depth":         stops,
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
