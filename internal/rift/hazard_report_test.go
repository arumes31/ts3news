package rift

import (
	"math"
	"testing"
)

func TestHazardSafetyReportReturnsFailingIntervals(t *testing.T) {
	arena := Arena{Hazards: []Hazard{{Obstacle: Obstacle{X: 705, Y: 315, W: 1, H: 175}, Kind: "fire", Period: 4, Duration: 1}}}
	failures, err := AuditHazardSafety(arena, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(failures) != 2 {
		t.Fatalf("intervals=%+v", failures)
	}
	for i, failure := range failures {
		if math.Abs(failure.From-(1.2+float64(i)*4)) > 1e-9 || math.Abs(failure.Until-(2.2+float64(i)*4)) > 1e-9 || len(failure.ActiveHazards) != 1 || failure.ActiveHazards[0] != 0 {
			t.Fatalf("interval=%+v", failure)
		}
	}
	arena.Hazards[0].Disabled = true
	if failures, err = AuditHazardSafety(arena, 8); err != nil || len(failures) != 0 {
		t.Fatalf("disabled hazard: %+v %v", failures, err)
	}
	arena.Hazards[0].Disabled = false
	arena.Hazards[0].Period = 1
	if _, err = AuditHazardSafety(arena, 8); err == nil {
		t.Fatal("invalid period accepted")
	}
	for _, horizon := range []float64{0, -1, math.Inf(1), 3601} {
		if _, err = AuditHazardSafety(Arena{}, horizon); err == nil {
			t.Fatal("invalid horizon accepted")
		}
	}
}

func TestHazardSafetyReportClipsOffsetsAndTracksOverlap(t *testing.T) {
	barrier := Hazard{Obstacle: Obstacle{X: 705, Y: 315, W: 1, H: 175}, Period: 4, Duration: 1, Offset: 1.5}
	intervals, err := AuditHazardSafety(Arena{Hazards: []Hazard{barrier}}, 2)
	if err != nil || len(intervals) != 1 || intervals[0].From != 0 || math.Abs(intervals[0].Until-.7) > 1e-9 {
		t.Fatalf("clipped: %+v %v", intervals, err)
	}
	barrier.Offset = -2
	intervals, err = AuditHazardSafety(Arena{Hazards: []Hazard{barrier}}, 5)
	if err != nil || len(intervals) != 1 || math.Abs(intervals[0].From-3.2) > 1e-9 || math.Abs(intervals[0].Until-4.2) > 1e-9 {
		t.Fatalf("delayed: %+v %v", intervals, err)
	}
	barrier.Offset = 0
	second := barrier
	second.Offset = .5
	intervals, err = AuditHazardSafety(Arena{Hazards: []Hazard{barrier, second}}, 3)
	if err != nil || len(intervals) != 3 {
		t.Fatalf("overlap: %+v %v", intervals, err)
	}
	for i, want := range []int{1, 2, 1} {
		if len(intervals[i].ActiveHazards) != want {
			t.Fatalf("active hazards: %+v", intervals)
		}
	}
}
