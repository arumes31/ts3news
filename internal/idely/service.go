package idely

import (
	"context"
	"errors"
	"time"
)

var errNoTracks = errors.New("idely: no lo-fi tracks configured")

// Snapshotter returns the current set of clients on the server. The production
// implementation wraps a ClientQuery session (SnapshotClientQuery); tests inject
// a fake.
type Snapshotter func() ([]Client, error)

// Service drives the Engine: it polls a Snapshotter on an interval and, when a
// talk-start feed is provided, reacts to talking immediately. It owns no
// connections itself — the runner supplies the Snapshotter and TalkStarts — so
// the loop logic is unit-testable.
type Service struct {
	Engine     *Engine
	Snapshot   Snapshotter
	TalkStarts <-chan int // optional real-time "clid started talking" feed
	// TalkCheck, if set, is polled once per tick for each active bot handle; it
	// returns true when a real user is talking in that bot's channel (via the
	// TS3AudioBot voice sensor). Such channels are stopped. This is how Idely
	// achieves "stop when someone talks", which ClientQuery cannot observe.
	TalkCheck    func(handle int) bool
	PollInterval time.Duration // default 5s
	Logf         func(format string, args ...any)
}

func (s *Service) logf(format string, args ...any) {
	if s.Logf != nil {
		s.Logf(format, args...)
	}
}

// tick takes one snapshot and lets the engine react. Snapshot errors are logged
// and skipped (transient ClientQuery hiccups shouldn't tear down the loop).
func (s *Service) tick() {
	clients, err := s.Snapshot()
	if err != nil {
		s.logf("idely: snapshot failed: %v", err)
		return
	}
	s.Engine.OnSnapshot(clients)

	// Stop any serenaded channel where a real user is now talking (heard by the
	// audio bot in that channel, since the headless detector can't hear voice).
	if s.TalkCheck != nil {
		for cid, handle := range s.Engine.Sessions() {
			if s.TalkCheck(handle) {
				s.Engine.StopForTalk(cid)
			}
		}
	}
}

// handleTalk reacts to a talk-start event. The engine's fast path stops playback
// if the talker is in the serenaded channel; if the clid is unknown (stale
// snapshot) we fall back to an immediate fresh snapshot.
func (s *Service) handleTalk(clid int) {
	if !s.Engine.OnTalkStart(clid) {
		s.tick()
	}
}

// Run drives the loop until ctx is cancelled, then shuts the engine down (so the
// audio bot doesn't keep playing after Idely exits).
func (s *Service) Run(ctx context.Context) {
	interval := s.PollInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()

	s.tick() // react immediately rather than waiting a full interval

	for {
		select {
		case <-ctx.Done():
			s.Engine.Shutdown()
			return
		case <-t.C:
			s.tick()
		case clid, ok := <-s.TalkStarts:
			if !ok {
				s.TalkStarts = nil // feed closed; keep polling
				continue
			}
			s.handleTalk(clid)
		}
	}
}
