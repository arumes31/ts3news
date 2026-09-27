package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func summonBossRun() *Run {
	r := ringRun()
	r.Enemies[0].HP = 1000
	r.Enemies[0].MaxHP = 1000
	r.Enemies[0].Armor = 0
	r.EncounterPlan = [][]Actor{{{Name: "Frozen catalog minion", ArtKey: "monster:frozen", Kind: "goblin", HP: 60, MaxHP: 60, Damage: 14, Speed: 75}}}
	return r
}
func TestBossSummonIntermissionSavedBoundedAndCleanup(t *testing.T) {
	r := summonBossRun()
	r.hurtEnemy(0, 550, "hit")
	if r.Enemies[0].SummonTimer != 4 {
		t.Fatal("phase has no intermission")
	}
	r.enemyTick(0, .9)
	if len(r.Enemies) != 1 {
		t.Fatal("summons arrived before warning")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	saved.enemyTick(0, .11)
	if len(saved.Enemies) != 3 {
		t.Fatalf("want two summons, got %d", len(saved.Enemies)-1)
	}
	for _, e := range saved.Enemies[1:] {
		if e.SummonOwner != "void-lord" || !e.Summoned || e.ArtKey != "monster:frozen" || e.ArrivalVulnerability != .85 {
			t.Fatalf("bad canonical summon: %+v", e)
		}
	}
	saved.hurtEnemy(0, 201, "hit")
	for j := 0; j < 50; j++ {
		saved.enemyTick(0, .1)
	}
	if len(saved.Enemies) != 3 {
		t.Fatal("phase replay bypassed population cap")
	}
	saved.hurtEnemy(0, 10000, "hit")
	for _, e := range saved.Enemies[1:] {
		if e.HP != 0 || e.Pose != "escape" {
			t.Fatal("victory left a hostile summon")
		}
	}
	if saved.Stats.Kills != 1 || saved.Stats.Bosses != 1 || len(saved.Drops) != 1 {
		t.Fatal("dismissal granted rewards")
	}
}
func TestBossSummonRespectsRoomPopulationAndDefeatBeforeArrival(t *testing.T) {
	r := summonBossRun()
	for j := 0; j < 7; j++ {
		r.Enemies = append(r.Enemies, Actor{ID: "other", Kind: "goblin", HP: 10})
	}
	r.hurtEnemy(0, 550, "hit")
	r.enemyTick(0, 1.1)
	if len(r.Enemies) != 8 {
		t.Fatal("room population exceeded")
	}
	r = summonBossRun()
	r.hurtEnemy(0, 550, "hit")
	r.hurtEnemy(0, 10000, "hit")
	r.enemyTick(0, 2)
	if len(r.Enemies) != 1 {
		t.Fatal("dead boss summoned")
	}
}

func TestBossSummonShotsRetireDuringProjectilePass(t *testing.T) {
	for _, friendlyFirst := range []bool{false, true} {
		r := summonBossRun()
		r.Player.X = 300
		r.Enemies[0].X = 500
		r.Enemies[0].Cooldown = 2
		r.Enemies = append(r.Enemies, Actor{ID: "minion", Kind: "archer", SummonOwner: "void-lord", Summoned: true, HP: 100, MaxHP: 100, X: 800, Y: 410, Cooldown: 2}, Actor{ID: "unrelated", Kind: "archer", HP: 100, X: 900, Y: 410, Cooldown: 2})
		r.Projectiles = []Projectile{{ID: 1, OwnerID: "minion", Enemy: true, X: 600, Y: 410, Life: 2, Power: 10}, {ID: 2, X: 500, Y: 410, Life: 2, Power: 10000, Skill: Skill{Kind: "fire"}}, {ID: 3, OwnerID: "unrelated", Enemy: true, X: 700, Y: 410, Life: 2, Power: 10}}
		if friendlyFirst {
			r.Projectiles[0], r.Projectiles[1] = r.Projectiles[1], r.Projectiles[0]
		}
		r.tick(Input{}, .01)
		if r.Enemies[0].HP != 0 || r.Enemies[1].HP != 0 || len(r.Projectiles) != 1 || r.Projectiles[0].OwnerID != "unrelated" {
			t.Fatal("mid-pass dismissal retained summon shots or lost unrelated shots")
		}
	}
}

func TestBossSummonWavesCannotRepeatAfterDefeats(t *testing.T) {
	r := summonBossRun()
	r.hurtEnemy(0, 550, "hit")
	r.enemyTick(0, 1.1)
	for i := 1; i < len(r.Enemies); i++ {
		r.hurtEnemy(i, 10000, "hit")
	}
	r.hurtEnemy(0, 201, "hit")
	r.enemyTick(0, 1.1)
	if len(r.Enemies) != 5 {
		t.Fatalf("expected two finite waves, got %d actors", len(r.Enemies))
	}
	ids := map[string]bool{}
	for _, a := range r.Enemies {
		if ids[a.ID] {
			t.Fatal("duplicate actor identity")
		}
		ids[a.ID] = true
	}
	for i := 1; i < len(r.Enemies); i++ {
		r.hurtEnemy(i, 10000, "hit")
	}
	for j := 0; j < 100; j++ {
		r.enemyTick(0, .1)
	}
	if len(r.Enemies) != 5 || r.Stats.Kills != 4 || len(r.Drops) != 4 {
		t.Fatal("repeated wave or duplicate minion rewards")
	}
}
func TestBossSummonPlacementAcrossCampaign(t *testing.T) {
	for _, level := range Campaign() {
		r := NewRunAtLevel("summon-placement", Build{HP: 10000}, time.Unix(0, 0), content.AbyssMobCatalog(), level.ID)
		r.Room = 2
		r.spawnRoom()
		r.Enemies[0].RingAttack = true
		r.Enemies[0].SummonTimer = 4
		r.Enemies[0].SummonPhase = 2
		before := len(r.Enemies)
		r.enemyTick(0, 1.1)
		if len(r.Enemies) != before+2 {
			t.Fatalf("mission%d did not place two minions", level.ID)
		}
		for _, a := range r.Enemies[before:] {
			assertSpawnOutsideHazards(t, r.Arena(), a)
			originalX, originalY := a.X, a.Y
			if !r.Arena().settleEnemySpawn(&a) || a.X != originalX || a.Y != originalY {
				t.Fatalf("mission%d spawned inside cover/drop", level.ID)
			}
			if a.ArtKey == "" || a.SummonOwner != r.Enemies[0].ID {
				t.Fatal("lost canonical identity/owner")
			}
		}
	}
}
