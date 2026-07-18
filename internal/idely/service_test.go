package idely

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestService_TickDrivesEngine(t *testing.T) {
	p := &fakePlayer{}
	e := newEngine(p)
	svc := &Service{
		Engine: e,
		Snapshot: func() ([]Client, error) {
			return []Client{voice(1, 10, "a", 20*time.Minute)}, nil
		},
	}
	svc.tick()
	if !e.Playing() {
		t.Fatal("tick should have started playback for an idle channel")
	}
}

func TestService_TickSwallowsSnapshotError(t *testing.T) {
	p := &fakePlayer{}
	e := newEngine(p)
	svc := &Service{
		Engine:   e,
		Snapshot: func() ([]Client, error) { return nil, errors.New("clientquery down") },
	}
	svc.tick()
	if e.Playing() {
		t.Fatal("snapshot error should not start playback")
	}
}

func TestService_HandleTalkFallsBackToSnapshot(t *testing.T) {
	p := &fakePlayer{}
	e := newEngine(p)
	e.OnSnapshot([]Client{voice(1, 10, "a", 20*time.Minute)})
	if !e.Playing() {
		t.Fatal("precondition: playing")
	}
	// Talk event for an unknown clid; the fallback snapshot shows that channel's
	// user is now active, so playback should stop.
	svc := &Service{
		Engine: e,
		Snapshot: func() ([]Client, error) {
			return []Client{voice(1, 10, "a", time.Second)}, nil
		},
	}
	svc.handleTalk(777)
	if e.Playing() {
		t.Fatal("fallback snapshot should have stopped playback")
	}
}

func TestService_RunStopsOnContextCancelAndShutsDown(t *testing.T) {
	p := &fakePlayer{}
	e := newEngine(p)
	svc := &Service{
		Engine:       e,
		PollInterval: 10 * time.Millisecond,
		Snapshot: func() ([]Client, error) {
			return []Client{voice(1, 10, "a", 20*time.Minute)}, nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { svc.Run(ctx); close(done) }()

	time.Sleep(50 * time.Millisecond)
	if !e.Playing() {
		t.Fatal("service should have started playing")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after cancel")
	}
	if e.Playing() {
		t.Fatal("Shutdown should have stopped playback on cancel")
	}
	if len(p.stopped) < 1 {
		t.Fatalf("expected a stop on shutdown, got %v", p.stopped)
	}
}
