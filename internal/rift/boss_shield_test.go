package rift

import (
	"encoding/json"
	"testing"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

func shellRun() *Run {
	r := testRun()
	a := AdaptMonster(content.Mob{Name: i18n.T("mob.kraken"), Type: content.MobBoss})
	a.ID = "kraken"
	a.X = 500
	a.Y = 410
	a.Armor = 0
	r.Enemies = []Actor{a}
	return r
}
func TestBossShieldProgressSavesAndBreaksOnce(t *testing.T) {
	r := shellRun()
	if r.Enemies[0].BossShieldHP != 90 || r.Enemies[0].BossShieldMax != 90 {
		t.Fatal("canonical Kraken has no shell")
	}
	hp := r.Enemies[0].HP
	r.hurtEnemy(0, 30, "hit")
	if r.Enemies[0].HP != hp || r.Enemies[0].BossShieldHP != 60 || r.Stats.DamageDealt != 0 {
		t.Fatal("shield damage counted as health damage")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	saved.Enemies[0].Windup = .1
	saved.Enemies[0].AttackName = "Rotating Fan"
	saved.hurtEnemy(0, 70, "hit")
	e := saved.Enemies[0]
	if e.BossShieldHP != 0 || e.HP != hp-10 || e.Windup != 0 || e.WeakPoint != 1.2 || e.Cooldown < 2 {
		t.Fatalf("invalid shell break %+v", e)
	}
	if saved.Stats.DamageDealt != 10 || len(saved.Drops) != 0 || saved.Stats.Kills != 0 {
		t.Fatal("shell rewards/stats")
	}
	saved.hurtEnemy(0, 1, "hit")
	count := 0
	for _, event := range saved.Events {
		if event.Kind == "boss_shield_break" {
			count++
		}
	}
	if count != 1 {
		t.Fatal("repeated shield break")
	}
}
func TestBossShieldUsesArmorAndZeroDamageCannotBreak(t *testing.T) {
	r := shellRun()
	r.Enemies[0].Armor = .5
	r.hurtEnemy(0, 40, "hit")
	if r.Enemies[0].BossShieldHP != 70 {
		t.Fatal("armor not applied before shell")
	}
	r.hurtEnemy(0, 0, "hit")
	if r.Enemies[0].BossShieldHP != 70 {
		t.Fatal("zero damage changed shell")
	}
	r.hurtEnemyPiercing(0, 40, "hit", 1)
	if r.Enemies[0].BossShieldHP != 30 {
		t.Fatal("piercing did not apply")
	}
	r.hurtEnemy(0, 10000, "hit")
	if r.Enemies[0].HP != 0 || len(r.Drops) != 1 || r.Stats.Kills != 1 {
		t.Fatal("lethal overflow failed")
	}
}

func TestBossShieldDoesNotEraseEarnedWeakPoint(t *testing.T) {
	r := shellRun()
	r.Enemies[0].WeakPoint = .8
	before := r.Enemies[0].HP
	r.hurtEnemy(0, 20, "hit")
	if r.Enemies[0].HP != before-25 || r.Enemies[0].BossShieldHP != 90 {
		t.Fatal("exposed weak point must bypass shell")
	}
	r.Enemies[0].WeakPoint = 0
	r.hurtEnemy(0, 20, "hit")
	if r.Enemies[0].HP != before-25 || r.Enemies[0].BossShieldHP != 70 {
		t.Fatal("expired weak point still bypasses shell")
	}
}
