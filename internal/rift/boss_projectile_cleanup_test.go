package rift

import (
	"encoding/json"
	"testing"
)

func TestBossDeathClearsOnlyItsProjectilesBeforeRoomClear(t *testing.T) {
	r := testRun()
	r.Player.X, r.Player.Y = 300, 410
	r.Enemies = []Actor{{ID: "boss", Kind: "boss", ArtKey: "monster:test", X: 450, Y: 410, HP: 100, MaxHP: 100, Windup: .01, Attacks: 1}, {ID: "archer", Kind: "archer", X: 650, Y: 410, HP: 100, MaxHP: 100, Windup: .01}}
	r.enemyTick(0, .02)
	r.enemyTick(1, .02)
	if len(r.Projectiles) != 2 {
		t.Fatal("fixture did not launch both shots")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	saved.hurtEnemy(0, 1000, "hit")
	hp := saved.Player.HP
	// Move the dead boss's shot into collision range before the next tick.
	saved.Projectiles[0].X, saved.Projectiles[0].Y = saved.Player.X, saved.Player.Y
	saved.tick(Input{}, .01)
	if saved.Player.HP != hp {
		t.Fatal("dead boss projectile still damaged player")
	}
	if len(saved.Projectiles) != 1 || saved.Projectiles[0].X < 600 {
		t.Fatal("cleanup removed surviving archer shot or retained boss shot")
	}
	if saved.Status != "fighting" {
		t.Fatal("test incorrectly relied on room-clear cleanup")
	}
}

func TestBossProjectileCleanupWhenFriendlyShotKillsMidPass(t *testing.T) {
	for _, friendlyFirst := range []bool{false, true} {
		r := testRun()
		r.Player.X, r.Player.Y = 300, 410
		r.Enemies = []Actor{{ID: "boss", Kind: "boss", ArtKey: "monster:test", X: 500, Y: 410, HP: 100, MaxHP: 100, Cooldown: 2}, {ID: "archer", Kind: "archer", X: 800, Y: 410, HP: 100, MaxHP: 100, Cooldown: 2}}
		bossShot := Projectile{ID: 1, OwnerID: "boss", Enemy: true, X: 600, Y: 410, Life: 2, Power: 10}
		friendly := Projectile{ID: 2, X: 500, Y: 410, Life: 2, Power: 1000, Skill: Skill{Kind: "fire"}}
		archerShot := Projectile{ID: 3, OwnerID: "archer", Enemy: true, X: 700, Y: 410, Life: 2, Power: 10}
		r.Projectiles = []Projectile{bossShot, friendly, archerShot}
		if friendlyFirst {
			r.Projectiles[0], r.Projectiles[1] = r.Projectiles[1], r.Projectiles[0]
		}
		r.tick(Input{}, .01)
		if r.Enemies[0].HP != 0 || len(r.Projectiles) != 1 || r.Projectiles[0].OwnerID != "archer" {
			t.Fatal("mid-pass boss death retained boss shots or lost another owner's shot")
		}
	}
}
