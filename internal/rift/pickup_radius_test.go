package rift

import (
	"testing"
	"time"
)

func TestPracticePickupIndicatorMatchesCollectionBoundary(t *testing.T) {
	now := time.Unix(100, 0)
	r, err := NewPracticeRun("radius", testRun().Build, "skills", now)
	if err != nil {
		t.Fatal(err)
	}
	radius := r.Practice.PickupRadius
	if radius <= 0 {
		t.Fatal("missing training radius")
	}
	r.Drops = []Drop{{ID: "inside", X: r.Player.X + radius - .01, Y: r.Player.Y}, {ID: "edge", X: r.Player.X + radius, Y: r.Player.Y}, {ID: "diagonal", X: r.Player.X + 50, Y: r.Player.Y + 50}}
	r.Step(Input{}, now.Add(50*time.Millisecond))
	if !r.Drops[0].Collected || r.Drops[1].Collected || r.Drops[2].Collected {
		t.Fatal("training circle differs from actual circular pickup boundary")
	}
}
