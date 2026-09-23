package rift

import "testing"

func TestTreasureEscapesAtBothArenaBoundaries(t *testing.T) {
	for _, right := range []bool{false, true} {
		r := testRun()
		r.Player.X = 200
		r.Player.Y = 410
		x := 56.0
		if right {
			x = Width - 56
			r.Player.X = Width - 200
		}
		r.Enemies = []Actor{{ID: "runner", Kind: "treasure", X: x, Y: 410, HP: 100, MaxHP: 100, Speed: 100}}
		r.Marked = "runner"
		r.enemyTick(0, .1)
		if r.Enemies[0].HP != 0 || r.Enemies[0].Pose != "escape" {
			t.Fatalf("right=%v: treasure did not escape", right)
		}
		if r.Marked != "" || r.Stats.Kills != 0 || r.Stats.TreasureGoblins != 0 || len(r.Drops) != 0 {
			t.Fatal("escape granted credit or retained target mark")
		}
		count := 0
		for _, e := range r.Events {
			if e.Kind == "treasure_escape" {
				count++
			}
		}
		if count != 1 {
			t.Fatal("expected one escape event")
		}
		events := len(r.Events)
		r.enemyTick(0, .1)
		if len(r.Events) != events {
			t.Fatal("escaped actor acted again")
		}
	}
}

func TestTreasureInsideBoundaryRemainsChaseable(t *testing.T) {
	r := testRun()
	r.Player.X = 200
	r.Player.Y = 410
	r.Enemies = []Actor{{ID: "runner", Kind: "treasure", X: 100, Y: 410, HP: 100, MaxHP: 100, Speed: 100}}
	r.enemyTick(0, .1)
	if r.Enemies[0].HP != 100 || r.Enemies[0].X >= 100 || r.Enemies[0].Pose != "run" {
		t.Fatal("treasure escaped before reaching boundary")
	}
}
