package rift

import (
	"encoding/json"
	"testing"
)

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

func TestTreasureApproachesEdgeWithoutOscillation(t *testing.T) {
	for _, right := range []bool{false, true} {
		r := testRun()
		x, direction := 180.0, -1.0
		if right {
			x, direction = Width-180, 1
		}
		r.Player.X, r.Player.Y = x-direction*120, 410
		r.Enemies = []Actor{{ID: "runner", Kind: "treasure", X: x, Y: 410, HP: 100, Speed: 100}}
		steps := 0
		for ; steps < 100 && r.Enemies[0].HP > 0; steps++ {
			before := r.Enemies[0].X
			r.enemyTick(0, .05)
			if (r.Enemies[0].X-before)*direction <= 0 {
				t.Fatal("treasure reversed or stalled while approaching escape edge")
			}
		}
		if steps < 2 || r.Enemies[0].Pose != "escape" {
			t.Fatal("treasure did not flee continuously to the edge")
		}
		escapedX := r.Enemies[0].X
		for i := 0; i < 20; i++ {
			r.Player.X = escapedX + float64(2*(i%2)-1)*20
			r.enemyTick(0, .05)
			if r.Enemies[0].X != escapedX || r.Enemies[0].Pose != "escape" {
				t.Fatal("crossing escaped treasure reactivated edge oscillation")
			}
		}
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var restored Run
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatal(err)
		}
		restored.enemyTick(0, .1)
		if restored.Enemies[0].HP != 0 || restored.Enemies[0].X != escapedX || restored.Enemies[0].Pose != "escape" {
			t.Fatal("saved escaped treasure resumed moving")
		}
		count := 0
		for _, event := range restored.Events {
			if event.Kind == "treasure_escape" {
				count++
			}
		}
		if count != 1 || restored.Stats.Kills != 0 || len(restored.Drops) != 0 {
			t.Fatal("escape was replayed or rewarded")
		}
	}
}

func TestTreasureKeepsFleeingAfterSaveAndHitRecovery(t *testing.T) {
	r := testRun()
	r.Player.X, r.Player.Y = 300, 410
	r.Enemies = []Actor{{ID: "runner", Kind: "treasure", X: 180, Y: 410, HP: 100, Speed: 100}}
	r.enemyTick(0, .05)
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	restored.Player.X = 500
	restored.Enemies[0].Pose = "hit"
	restored.Enemies[0].PoseTime = .01
	before := restored.Enemies[0].X
	restored.enemyTick(0, .05)
	if !restored.Enemies[0].Fleeing || restored.Enemies[0].X >= before {
		t.Fatal("treasure resumed pursuit after saved flee state and hit recovery")
	}
}
