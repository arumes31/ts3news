package rift

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"ts3news/internal/content"
)

func woundedEliteRun(kind string, hp float64) *Run {
	r := testRun()
	r.Player.X, r.Player.Y = 500, 410
	r.Enemies = []Actor{{ID: "elite", Name: "Elite probe", Kind: kind, Elite: true, X: 700, Y: 410, HP: hp, MaxHP: 100, Speed: 80, Damage: 25, Facing: -1}}
	return r
}

func TestWoundedMeleeElitePrioritizesTelegraphedCharge(t *testing.T) {
	for _, hp := range []float64{36, 35} {
		r := woundedEliteRun("knight", hp)
		r.enemyTick(0, .02)
		e := r.Enemies[0]
		if hp > 35 {
			if e.Enraged || e.AttackName == "Charge" {
				t.Fatal("healthy elite changed priority")
			}
			continue
		}
		if !e.Enraged || !e.Charging || e.AttackName != "Charge" || e.Windup != .7 || e.Damage != 25 {
			t.Fatal("wounded melee elite lacks warned charge")
		}
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved Run
		if err = json.Unmarshal(raw, &saved); err != nil {
			t.Fatal(err)
		}
		saved.Events = nil
		saved.Enemies[0].HP = 80
		saved.enemyTick(0, .02)
		if !saved.Enemies[0].Enraged || !saved.Enemies[0].Charging || findEvent(saved.Events, "elite_desperation") != nil {
			t.Fatal("saved phase reset or replayed cue")
		}
	}
}

func TestWoundedRangedElitePrioritizesMoreSpaceButStillFires(t *testing.T) {
	healthy := woundedEliteRun("archer", 36)
	healthy.enemyTick(0, .02)
	if healthy.Enemies[0].Windup == 0 {
		t.Fatal("healthy ranged control failed to aim")
	}
	r := woundedEliteRun("archer", 35)
	r.enemyTick(0, .02)
	if !r.Enemies[0].Enraged || r.Enemies[0].X <= 700 || r.Enemies[0].Windup != 0 {
		t.Fatal("wounded ranged elite did not prioritize distance")
	}
	for i := 0; i < 80 && len(r.Projectiles) == 0; i++ {
		r.enemyTick(0, .02)
	}
	if len(r.Projectiles) == 0 || r.Enemies[0].X < 720 || r.Enemies[0].X > 724 {
		t.Fatal("wounded ranged elite failed to stop retreating and shoot")
	}
}

func TestWoundedElitePreservesCommittedAttackAndPinnedFallback(t *testing.T) {
	r := woundedEliteRun("archer", 35)
	r.Enemies[0].Windup = .01
	r.enemyTick(0, .02)
	if len(r.Projectiles) != 1 || r.Enemies[0].X != 700 {
		t.Fatal("phase change interrupted committed shot")
	}
	r = woundedEliteRun("archer", 35)
	r.Enemies[0].X = 1565
	r.Player.X = 1365
	r.enemyTick(0, .02)
	if r.Enemies[0].Windup == 0 {
		t.Fatal("pinned wounded archer never attacks")
	}
}

func TestWoundedEliteExclusionsAndCanonicalTiers(t *testing.T) {
	for _, kind := range []string{"boss", "treasure"} {
		r := woundedEliteRun(kind, 20)
		r.enemyTick(0, .02)
		if r.Enemies[0].Enraged {
			t.Fatal("specialized role entered elite phase")
		}
	}
	r := woundedEliteRun("knight", 20)
	r.Enemies[0].Elite = false
	r.enemyTick(0, .02)
	if r.Enemies[0].Enraged {
		t.Fatal("ordinary fighter entered elite phase")
	}
	for _, mob := range content.AbyssMobCatalog() {
		a := AdaptMonster(mob)
		if mob.Type == content.MobEliteMinion || mob.Type == content.MobElite || mob.Type == content.MobMiniboss {
			if !a.Elite {
				t.Fatalf("missing canonical elite flag: %s", mob.Name)
			}
		}
	}
}

func TestWoundedElitesEngageFromAuthoredSpawns(t *testing.T) {
	checked := 0
	for _, level := range Campaign() {
		base := NewRunAtLevel("pursuit-audit", Build{HP: 10000}, time.Unix(0, 0), content.AbyssMobCatalog(), level.ID)
		for room, spawns := range base.EncounterPlan {
			for index, spawn := range spawns {
				if !spawn.Elite || spawn.Kind == "boss" || spawn.Kind == "treasure" {
					continue
				}
				t.Run(fmt.Sprintf("%d/%d/%d", level.ID, room, index), func(t *testing.T) {
					r := NewRunAtLevel("wounded-route", Build{HP: 10000}, time.Unix(0, 0), content.AbyssMobCatalog(), level.ID)
					r.Room = room
					r.spawnRoom()
					r.RoomObjective = nil
					spawn.HP = spawn.MaxHP * .35
					spawn.Patrol = false
					spawn.Alerted = true
					r.Enemies = []Actor{spawn}
					r.Projectiles = nil
					r.PackAttackLockout = 0
					hp := r.Player.HP
					engaged := false
					for frame := 0; frame < 3000; frame++ {
						r.enemyTick(0, .02)
						if r.Player.HP < hp || len(r.Projectiles) > 0 && r.clearProjectilePath(&r.Enemies[0], &r.Player) {
							engaged = true
							break
						}
					}
					if !engaged {
						e := r.Enemies[0]
						t.Fatalf("wounded %s stuck from %.1f,%.1f at %.1f,%.1f; attacks %d", e.Name, spawn.X, spawn.Y, e.X, e.Y, e.Attacks)
					}
					checked++
				})
			}
		}
	}
	if checked == 0 {
		t.Fatal("no wounded elite routes checked")
	}
	t.Logf("Verified %d wounded elite spawn routes", checked)
}

func TestWoundedExistingChargerPrioritizesChargeAtMoreRanges(t *testing.T) {
	for _, x := range []float64{590, 850} {
		for _, hp := range []float64{36, 35} {
			r := woundedEliteRun("knight", hp)
			r.Enemies[0].Charging = true
			r.Enemies[0].X = x
			r.enemyTick(0, .02)
			if (r.Enemies[0].AttackName == "Charge") != (hp == 35) {
				t.Fatalf("HP %v at %v did not change existing charger priorities", hp, x)
			}
		}
	}
}
