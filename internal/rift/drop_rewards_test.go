package rift

import (
	"testing"
	"time"
)

func TestEnemyDropGoldMatchesActualDefeatRewards(t *testing.T) {
	for room, want := range []int64{15, 30, 45} {
		run := NewRunWithCatalog("reward", Build{HP: 300}, time.Unix(100, 0), nil)
		run.Room = room
		run.Enemies = []Actor{{ID: "reward-probe", Kind: "goblin", HP: 1, MaxHP: 1, X: 400, Y: 400}}
		run.hurtEnemy(0, 100, "hit")
		if len(run.Drops) != 1 || run.Drops[0].Gold != want || EnemyDropGold(room) != want {
			t.Fatalf("room %d drops=%+v", room, run.Drops)
		}
	}
	if EnemyDropGold(-1) != 0 || EnemyDropGold(len(Rooms)) != 0 {
		t.Fatal("invalid room has a reward")
	}
}
