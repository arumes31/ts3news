package rift

import "testing"

func TestReachabilityReportIdentifiesBlockedSideAndStart(t *testing.T) {
	start := Actor{ID: "player", X: 160, Y: 410}
	targets := []Actor{{ID: "near", X: 200, Y: 410}, {ID: "far", X: 900, Y: 410}}
	arena := Arena{HighCover: []Obstacle{{X: 500, Y: 250, W: 40, H: 300}}}
	failures := AuditArenaReachability(arena, start, targets)
	if len(failures) != 1 || failures[0].ActorID != "far" || failures[0].X != 900 || failures[0].Y != 410 || failures[0].Reason != "unreachable" {
		t.Fatalf("failures=%+v", failures)
	}
	if failures = AuditArenaReachability(Arena{}, start, targets); len(failures) != 0 {
		t.Fatalf("open arena: %+v", failures)
	}
	start.X = 510
	if failures = AuditArenaReachability(arena, start, targets); len(failures) != 1 || failures[0].Reason != "blocked_start" {
		t.Fatalf("blocked start: %+v", failures)
	}
}
