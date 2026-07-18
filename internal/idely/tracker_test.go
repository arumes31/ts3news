package idely

import (
	"testing"
	"time"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time      { return c.t }
func (c *fakeClock) add(d time.Duration) { c.t = c.t.Add(d) }

func TestTracker_IdleGrowsWithoutActivity(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1000, 0)}
	tr := NewActivityTracker(clk.now)

	// First sighting: idle 0.
	out := tr.Observe([]Client{voice(1, 10, "a", 0)})
	if out[0].Idle != 0 {
		t.Fatalf("first sighting idle = %v, want 0", out[0].Idle)
	}
	// 20 minutes pass with no activity.
	clk.add(20 * time.Minute)
	out = tr.Observe([]Client{voice(1, 10, "a", 0)})
	if out[0].Idle != 20*time.Minute {
		t.Fatalf("idle = %v, want 20m", out[0].Idle)
	}
}

func TestTracker_TouchResetsIdle(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1000, 0)}
	tr := NewActivityTracker(clk.now)
	tr.Observe([]Client{voice(1, 10, "a", 0)})
	clk.add(20 * time.Minute)

	tr.Touch(1) // the client talked
	out := tr.Observe([]Client{voice(1, 10, "a", 0)})
	if out[0].Idle != 0 {
		t.Fatalf("after touch idle = %v, want 0", out[0].Idle)
	}
}

func TestTracker_ChannelMoveIsActivity(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1000, 0)}
	tr := NewActivityTracker(clk.now)
	tr.Observe([]Client{voice(1, 10, "a", 0)})
	clk.add(20 * time.Minute)

	// Client moved to channel 20 => counts as active now.
	out := tr.Observe([]Client{voice(1, 20, "a", 0)})
	if out[0].Idle != 0 {
		t.Fatalf("after move idle = %v, want 0", out[0].Idle)
	}
}

func TestTracker_PrunesDepartedClients(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1000, 0)}
	tr := NewActivityTracker(clk.now)
	tr.Observe([]Client{voice(1, 10, "a", 0)})
	tr.Observe([]Client{}) // client 1 left
	// Client 1 returns; should be treated as freshly active, not stale.
	clk.add(20 * time.Minute)
	out := tr.Observe([]Client{voice(1, 10, "a", 0)})
	if out[0].Idle != 0 {
		t.Fatalf("returning client idle = %v, want 0 (was pruned)", out[0].Idle)
	}
}

// End-to-end: tracker + detector make a quiet channel idle, and a talk resets it.
func TestTracker_FeedsDetector(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1000, 0)}
	tr := NewActivityTracker(clk.now)
	c := cfg() // 15m threshold

	clients := []Client{voice(1, 10, "a", 0), voice(2, 10, "b", 0)}
	tr.Observe(clients)
	if got := c.IdleChannels(tr.Observe(clients)); len(got) != 0 {
		t.Fatalf("channel should not be idle yet, got %v", got)
	}
	clk.add(16 * time.Minute) // everyone quiet 16m
	if got := c.IdleChannels(tr.Observe(clients)); len(got) != 1 || got[0] != 10 {
		t.Fatalf("channel should be idle after 16m quiet, got %v", got)
	}
	tr.Touch(2) // b talks
	if got := c.IdleChannels(tr.Observe(clients)); len(got) != 0 {
		t.Fatalf("talk should make channel active, got %v", got)
	}
}
