package rift

import (
	"fmt"
	"testing"
	"time"

	"ts3news/internal/content"
)

// Isolate locomotion from objective work, alert acquisition and crowd scheduling.
// EncounterPlan includes later-wave entries, which may not yet be in Enemies.
func TestPursuitFromEveryAuthoredCampaignSpawn(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	checked := 0
	seen := map[string]bool{}
	for _, level := range Campaign() {
		base := NewRunAtLevel("pursuit-audit", Build{HP: 10000}, time.Unix(0, 0), catalog, level.ID)
		for room, spawns := range base.EncounterPlan {
			for index, spawn := range spawns {
				t.Run(fmt.Sprintf("%d/%d/%d/%s", level.ID, room, index, spawn.Kind), func(t *testing.T) {
					r := NewRunAtLevel("pursuit-audit", Build{HP: 10000}, time.Unix(0, 0), catalog, level.ID)
					r.Room = room
					r.spawnRoom()
					r.RoomObjective = nil
					spawn.Patrol = false
					spawn.Alerted = true
					r.Enemies = []Actor{spawn}
					r.Projectiles = nil
					r.PackAttackLockout = 0
					hp := r.Player.HP
					reached := false
					for step := 0; step < 3000; step++ {
						r.enemyTick(0, .02)
						e := &r.Enemies[0]
						if e.Kind == "treasure" {
							reached = e.HP <= 0
						} else {
							reached = r.Player.HP < hp || len(r.Projectiles) > 0 && r.clearProjectilePath(e, &r.Player)
						}
						if reached {
							break
						}
					}
					checked++
					seen[spawn.ArtKey] = true
					if !reached {
						e := r.Enemies[0]
						t.Fatalf("%s failed from %.1f,%.1f to entry %.1f,%.1f; ended %.1f,%.1f route %.1f,%.1f windup %.2f attacks %d", spawn.Name, spawn.X, spawn.Y, r.Player.X, r.Player.Y, e.X, e.Y, e.RouteX, e.RouteY, e.Windup, e.Attacks)
					}
				})
			}
		}
	}
	for _, mob := range catalog {
		if !seen["monster:"+mob.Name] {
			t.Errorf("canonical template missing from audit: %s", mob.Name)
		}
	}
	t.Logf("Checked %d authored spawns covering %d canonical templates", checked, len(seen))
}

// Every position also receives each locomotion/attack role. Production settlement
// applies the role's footprint before pursuit; no wall or drop geometry is removed.
func TestEveryAuthoredSpawnSupportsAllCombatRoles(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	checked := 0
	for _, level := range Campaign() {
		base := NewRunAtLevel("pursuit-audit", Build{HP: 10000}, time.Unix(0, 0), catalog, level.ID)
		for room, spawns := range base.EncounterPlan {
			for index, spawn := range spawns {
				for _, kind := range []string{"goblin", "knight", "archer", "boss", "treasure", "wolf", "spore"} {
					t.Run(fmt.Sprintf("%d/%d/%d/%s", level.ID, room, index, kind), func(t *testing.T) {
						r := NewRunAtLevel("pursuit-audit", Build{HP: 10000}, time.Unix(0, 0), catalog, level.ID)
						r.Room = room
						r.spawnRoom()
						r.RoomObjective = nil
						e := Actor{ID: "role-probe", Name: kind, Kind: kind, X: spawn.X, Y: spawn.Y, HP: 100, MaxHP: 100, Speed: 90, Damage: 20, Phase: 1, Facing: -1, Alerted: true}
						if !r.Arena().settleEnemySpawn(&e) {
							t.Fatal("no safe spawn for role")
						}
						r.Enemies = []Actor{e}
						r.Projectiles = nil
						r.PackAttackLockout = 0
						hp := r.Player.HP
						reached := false
						for step := 0; step < 3000; step++ {
							r.enemyTick(0, .02)
							e = r.Enemies[0]
							if kind == "treasure" {
								reached = e.HP <= 0
							} else {
								reached = r.Player.HP < hp || len(r.Projectiles) > 0 && r.clearProjectilePath(&e, &r.Player)
							}
							if reached {
								break
							}
						}
						checked++
						if !reached {
							t.Fatalf("%s failed from %.1f,%.1f; ended %.1f,%.1f route %.1f,%.1f", kind, spawn.X, spawn.Y, e.X, e.Y, e.RouteX, e.RouteY)
						}
					})
				}
			}
		}
	}
	t.Logf("Checked %d authored spawn / role combinations", checked)
}
