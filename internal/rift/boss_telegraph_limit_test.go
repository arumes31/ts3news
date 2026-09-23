package rift

import (
	"encoding/json"
	"testing"
)

func TestBossTelegraphsQueueWithoutBlockingRegularEnemies(t *testing.T) {
	r := testRun()
	r.Player.X, r.Player.Y = 500, 410
	r.Level = &Level{Rooms: []Arena{{MaxAttackers: 4}}}
	r.Enemies = []Actor{{ID: "boss-a", Kind: "boss", X: 550, Y: 410, HP: 100}, {ID: "boss-b", Kind: "boss", X: 560, Y: 410, HP: 100}, {ID: "goblin", Kind: "goblin", X: 540, Y: 410, HP: 100}}
	for i := range r.Enemies {
		r.enemyTick(i, .01)
	}
	if r.Enemies[0].Windup == 0 || r.Enemies[1].Windup != 0 {
		t.Fatal("multiple boss telegraphs overlapped")
	}
	if r.Enemies[2].Windup == 0 {
		t.Fatal("boss telegraph unnecessarily blocked ordinary enemy")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	saved.enemyTick(1, .01)
	if saved.Enemies[1].Windup != 0 {
		t.Fatal("save lost telegraph coordination")
	}
	saved.Enemies[0].Windup = .01
	saved.enemyTick(0, .02)
	saved.enemyTick(1, .01)
	if saved.Enemies[0].Attacks != 1 || saved.Enemies[1].Windup <= 0 {
		t.Fatal("queued boss did not start after preceding telegraph completed")
	}
}

func TestDefeatedBossDoesNotHoldTelegraphSlot(t *testing.T) {
	r := testRun()
	r.Player.X, r.Player.Y = 500, 410
	r.Enemies = []Actor{{ID: "dead", Kind: "boss", X: 550, Y: 410, HP: 0, Windup: 1}, {ID: "alive", Kind: "boss", X: 560, Y: 410, HP: 100}}
	r.enemyTick(1, .01)
	if r.Enemies[1].Windup == 0 {
		t.Fatal("dead boss retained telegraph slot")
	}
}
