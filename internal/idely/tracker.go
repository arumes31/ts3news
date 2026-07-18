package idely

import (
	"sync"
	"time"
)

// ActivityTracker maintains a per-client "last active" timestamp so Idely can
// measure idleness itself. This is necessary because TeamSpeak does NOT expose
// other clients' server-side client_idle_time over ClientQuery (it always reads
// 0 for remote clients) — the only activity signals a ClientQuery client gets
// are voice (notifytalkstatuschange) and view changes (join / channel move).
//
// So "idle" here means "no voice activity", which matches the feature's intent:
// play music when a channel has gone quiet, stop the moment someone talks.
//
// A consequence is a warm-up: Idely only knows a client has been quiet since it
// first saw them, so on startup it treats everyone as just-active and lets the
// idle threshold elapse before serenading.
//
// ActivityTracker is safe for concurrent use (the poll loop and the talk-event
// listener touch it from different goroutines).
type ActivityTracker struct {
	now func() time.Time
	mu  sync.Mutex
	// lastActive[clid] is when the client last talked, joined, or changed channel.
	lastActive map[int]time.Time
	lastCID    map[int]int
}

// NewActivityTracker builds a tracker. now defaults to time.Now.
func NewActivityTracker(now func() time.Time) *ActivityTracker {
	if now == nil {
		now = time.Now
	}
	return &ActivityTracker{
		now:        now,
		lastActive: map[int]time.Time{},
		lastCID:    map[int]int{},
	}
}

// Touch marks a client active as of now (called when it starts talking).
func (t *ActivityTracker) Touch(clid int) {
	t.mu.Lock()
	t.lastActive[clid] = t.now()
	t.mu.Unlock()
}

// Observe reconciles the tracker with a fresh client list and returns the same
// clients with their Idle duration filled in. Newly-seen clients and clients
// that changed channel are treated as active now; clients that vanished are
// pruned. The returned slice is the input slice with Idle mutated.
func (t *ActivityTracker) Observe(clients []Client) []Client {
	now := t.now()
	t.mu.Lock()
	defer t.mu.Unlock()

	present := make(map[int]bool, len(clients))
	for i := range clients {
		c := &clients[i]
		present[c.CLID] = true

		prevCID, known := t.lastCID[c.CLID]
		if !known || prevCID != c.CID {
			t.lastActive[c.CLID] = now // just appeared or moved => active
		} else if _, ok := t.lastActive[c.CLID]; !ok {
			t.lastActive[c.CLID] = now
		}
		t.lastCID[c.CLID] = c.CID
		c.Idle = now.Sub(t.lastActive[c.CLID])
	}

	for clid := range t.lastActive {
		if !present[clid] {
			delete(t.lastActive, clid)
			delete(t.lastCID, clid)
		}
	}
	return clients
}
