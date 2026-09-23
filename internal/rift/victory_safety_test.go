package rift

import (
	"testing"
	"time"
)

func TestEnemyWindupsCannotResolveOutsideActiveCombat(t *testing.T) {
	for _, state := range []string{"cleared", "complete", "paused"} {
		t.Run(state, func(t *testing.T) {
			r := testRun()
			r.Status = state
			if state == "paused" {
				r.Status = "fighting"
				r.Paused = true
			}
			r.Enemies = []Actor{{ID: "pending", Kind: "archer", X: r.Player.X + 40, Y: r.Player.Y, HP: 100, MaxHP: 100, Windup: .01}}
			before := r.Enemies[0]
			hp := r.Player.HP
			r.Step(Input{}, time.UnixMilli(r.LastMS+100))
			if r.Enemies[0] != before || len(r.Projectiles) != 0 || r.Player.HP != hp {
				t.Fatal("inactive combat advanced an enemy attack")
			}
		})
	}
}

func TestRoomClearRemovesInFlightHostileProjectiles(t *testing.T) {
	r := testRun()
	r.Enemies = nil
	r.Projectiles = []Projectile{{X: 1200, Y: 410, VX: -300, Enemy: true, Power: 20, Life: 4}}
	r.tick(Input{}, .01)
	if r.Status != "cleared" || len(r.Projectiles) != 0 {
		t.Fatal("room clear left a projectile active")
	}
	hp := r.Player.HP
	r.Step(Input{}, time.UnixMilli(r.LastMS+200))
	if r.Player.HP != hp {
		t.Fatal("victory transition dealt projectile damage")
	}
}
