package rift

import (
	"encoding/json"
	"math"
	"testing"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

func fanRun() *Run {
	r := testRun()
	r.Player.X, r.Player.Y = 500, 410
	e := AdaptMonster(content.Mob{Name: i18n.T("mob.kraken"), Type: content.MobBoss})
	e.ID, e.X, e.Y, e.Attacks = "kraken", 700, 410, 1
	r.Enemies = []Actor{e}
	r.Level = &Level{Rooms: []Arena{{}}}
	return r
}

func TestKrakenFanRotatesWithLockedAimAndSavedWarning(t *testing.T) {
	for volley := 0; volley < 4; volley++ {
		r := fanRun()
		r.Enemies[0].Attacks = volley*2 + 1
		plan := r.NextBossAttack(r.Enemies[0])
		if plan.Name != "Rotating Fan" || plan.Kind != "fan" || plan.Windup != 1.25 {
			t.Fatalf("missing fan plan: %+v", plan)
		}
		r.enemyTick(0, .02)
		e := r.Enemies[0]
		wantRotation := float64(volley%3-1) * .18
		if e.FanRotation != wantRotation || e.Windup != 1.25 {
			t.Fatal("wrong warned rotation")
		}
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved Run
		if err = json.Unmarshal(raw, &saved); err != nil {
			t.Fatal(err)
		}
		saved.Player.X, saved.Player.Y = 1100, 480
		saved.enemyTick(0, 1.25)
		if len(saved.Projectiles) != 5 {
			t.Fatalf("got %d shots", len(saved.Projectiles))
		}
		ids := map[int]bool{}
		for n, p := range saved.Projectiles {
			angle := math.Pi + wantRotation + float64(n-2)*.22
			if math.Abs(p.VX-math.Cos(angle)*260) > 1e-7 || math.Abs(p.VY-math.Sin(angle)*260) > 1e-7 {
				t.Fatal("shots did not match locked fan")
			}
			if p.OwnerID != "kraken" || !p.Enemy || ids[p.ID] || p.Power > math.Max(18, e.Damage)*.65+1e-9 {
				t.Fatal("invalid ownership, power or ID")
			}
			ids[p.ID] = true
		}
		if saved.Enemies[0].Cooldown != 2 {
			t.Fatal("missing fan recovery")
		}
	}
}

func TestBossFanPreservesOtherPatternsAndInterrupts(t *testing.T) {
	r := fanRun()
	r.Enemies[0].Attacks = 0
	if r.NextBossAttack(r.Enemies[0]).Kind != "slam" {
		t.Fatal("fan replaced slam")
	}
	r = fanRun()
	r.Enemies[0].VolleyFan = false
	if r.NextBossAttack(r.Enemies[0]).Kind != "projectile" {
		t.Fatal("ordinary volley changed")
	}
	r = fanRun()
	r.enemyTick(0, .02)
	r.Enemies[0].BossStagger = 80
	r.hurtEnemy(0, 1, "hit")
	for n := 0; n < 50; n++ {
		r.enemyTick(0, .02)
	}
	if len(r.Projectiles) != 0 {
		t.Fatal("guard-broken fan still fired")
	}
	r = fanRun()
	r.Practice = &PracticeState{SlowTelegraphs: true}
	r.enemyTick(0, .02)
	if r.Enemies[0].Windup != 2.5 {
		t.Fatal("fan ignored slow practice")
	}
}
