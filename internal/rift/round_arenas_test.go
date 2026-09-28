package rift

import (
	"encoding/json"
	"testing"
)

func TestRoundArenaGroundFootprintsAndSweeps(t *testing.T) {
	a := Arena{Round: &RoundArena{X: 800, Y: 402.5, RadiusX: 740, RadiusY: 87.5}}
	if !a.Round.valid() {
		t.Fatal("valid round arena rejected")
	}
	for _, tc := range []struct {
		x, y float64
		want bool
	}{{800, 402.5, true}, {800, 325, true}, {800, 320, false}, {60, 402.5, false}, {80, 402.5, true}, {100, 330, false}, {800, 480, true}, {800, 490, false}} {
		if got := a.groundPath(800, 402.5, tc.x, tc.y, 10); got != tc.want {
			t.Fatalf("endpoint %.1f %.1f: %v", tc.x, tc.y, got)
		}
	}
	if a.groundPath(100, 330, 800, 402.5, 10) {
		t.Fatal("invalid starting point accepted")
	}
	if !a.groundPath(300, 402.5, 1300, 402.5, 18) {
		t.Fatal("boss crossing rejected")
	}
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var saved Arena
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Round == nil || *saved.Round != *a.Round {
		t.Fatal("saved circular geometry lost")
	}
}

func TestRoundArenaMovementSpawnAndDetachedCopy(t *testing.T) {
	r := dropEdgeTestRun()
	r.Level.Rooms[0] = Arena{Round: &RoundArena{X: 800, Y: 402.5, RadiusX: 740, RadiusY: 87.5}}
	r.Player.X, r.Player.Y = 800, 402.5
	r.moveActor(&r.Player, 0, 100, false)
	if r.Player.Y != 402.5 {
		t.Fatal("movement left circular floor")
	}
	enemy := Actor{Kind: "boss", X: 1500, Y: 480}
	if !r.Arena().settleEnemySpawn(&enemy) || !r.Arena().groundPath(enemy.X, enemy.Y, enemy.X, enemy.Y, actorClearance(&enemy)) {
		t.Fatal("boss spawn outside circular floor")
	}
	original := *r.Level
	copied := cloneCampaignLevel(original)
	copied.Rooms[0].Round.RadiusX = 500
	if original.Rooms[0].Round.RadiusX != 740 {
		t.Fatal("round arena copy aliases source")
	}
}
