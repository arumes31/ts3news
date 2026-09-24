package rift

import "testing"

func TestChargesSurviveTierTransition(t *testing.T) {
	for charges := 0; charges <= 3; charges++ {
		r := testRun()
		r.Resource = charges
		r.Marked = "old-target"
		r.Status = "cleared"
		r.NextRoom()
		if r.Resource != charges {
			t.Fatalf("transition changed %d charges to %d", charges, r.Resource)
		}
		if r.Marked != "" {
			t.Fatal("old target mark survived")
		}
	}
}
