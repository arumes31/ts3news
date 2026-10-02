package rift

import (
	"encoding/json"
	"testing"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

func bossChargeRun() *Run {
	r := chargeRun()
	r.Enemies[0] = AdaptMonster(content.Mob{Name: i18n.T("mob.ancient_dragon"), Type: content.MobBoss})
	e := &r.Enemies[0]
	e.ID, e.X, e.Y, e.Attacks = "dragon", 150, 410, 2
	return r
}

func TestBossChargeWarnsLocksAimAndRecoversAtWall(t *testing.T) {
	r := bossChargeRun()
	r.Enemies[0].X = 700
	r.Player.X = 1000
	r.Level = &Level{Rooms: []Arena{{HighCover: []Obstacle{{820, 350, 40, 100}}}}}
	plan := r.NextBossAttack(r.Enemies[0])
	if plan.Kind != "charge" || plan.Windup != 1.15 || plan.Recovery != 2.3 {
		t.Fatalf("missing boss charge plan: %+v", plan)
	}
	r.enemyTick(0, .02)
	if r.Enemies[0].AttackName != "Charge" || r.Enemies[0].Windup != plan.Windup {
		t.Fatal("charge missing warning")
	}
	r.Player.Y = 480
	for n := 0; n < 150 && r.Enemies[0].ChargeRecovery == 0; n++ {
		r.enemyTick(0, .02)
	}
	e := &r.Enemies[0]
	if e.ChargeRecovery != 1.2 || e.Cooldown != 2.3 || e.X >= 802 || e.TargetY != 410 || r.Player.HP != r.Player.MaxHP {
		t.Fatalf("unsafe wall recovery: %+v", e)
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	x := saved.Enemies[0].X
	for n := 0; n < 50; n++ {
		saved.enemyTick(0, .02)
	}
	if saved.Enemies[0].X != x || saved.Enemies[0].ChargeRecovery <= 0 {
		t.Fatal("saved wall recovery did not remain stationary")
	}
	hp := saved.Enemies[0].HP
	saved.hurtEnemy(0, 5, "hit")
	if saved.Enemies[0].HP >= hp {
		t.Fatal("boss recovery cannot be punished")
	}
}

func TestBossChargeSequenceAndCommittedAttack(t *testing.T) {
	for _, attack := range []int{0, 1, 3, 4} {
		r := bossChargeRun()
		r.Enemies[0].Attacks = attack
		if r.NextBossAttack(r.Enemies[0]).Kind == "charge" {
			t.Fatal("charge replaced ordinary sequence")
		}
		r.enemyTick(0, .02)
		if r.Enemies[0].AttackName == "Charge" {
			t.Fatal("unexpected charge")
		}
	}
	r := bossChargeRun()
	r.Enemies[0].AttackName = "Committed slam"
	r.Enemies[0].Windup = .01
	r.enemyTick(0, .02)
	if findEvent(r.Events, "slam") == nil || r.Enemies[0].ChargeActive {
		t.Fatal("charge replaced committed attack")
	}
	r = bossChargeRun()
	r.Practice = &PracticeState{SlowTelegraphs: true}
	r.enemyTick(0, .02)
	if r.Enemies[0].Windup != 2.3 {
		t.Fatal("charge ignored slow practice")
	}
}

func TestBossChargeSaveWarningRushAndInterrupt(t *testing.T) {
	for _, active := range []bool{false, true} {
		r := bossChargeRun()
		r.enemyTick(0, .02)
		if active {
			for n := 0; n < 70 && !r.Enemies[0].ChargeActive; n++ {
				r.enemyTick(0, .02)
			}
		}
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved Run
		if err = json.Unmarshal(raw, &saved); err != nil {
			t.Fatal(err)
		}
		saved.Player.Jump = .5
		for n := 0; n < 150 && saved.Enemies[0].ChargeRecovery == 0; n++ {
			saved.enemyTick(0, .02)
		}
		if saved.Enemies[0].ChargeRecovery == 0 || saved.Player.HP != saved.Player.MaxHP {
			t.Fatal("saved charge failed jump counterplay")
		}
		r.hurtEnemy(0, 1, "hit")
		if r.Enemies[0].ChargeActive || r.Enemies[0].Windup != 0 || r.Enemies[0].ChargeRecovery != 1.2 {
			t.Fatal("damage failed to interrupt boss charge")
		}
	}
}

func TestLongBossChargeKeepsAttackerSlotUntilRushEnds(t *testing.T) {
	r := bossChargeRun()
	r.Level = &Level{Rooms: []Arena{{MaxAttackers: 1}}}
	r.Player.X = 630
	r.enemyTick(0, .02)
	r.Player.Y = 480
	for n := 0; n < 60 && !r.Enemies[0].ChargeActive; n++ {
		r.enemyTick(0, .02)
	}
	for n := 0; n < 55; n++ {
		r.enemyTick(0, .02)
	}
	if !r.Enemies[0].ChargeActive {
		t.Fatal("long charge finished early")
	}
	if r.canStartEnemyAttack(&Actor{Kind: "goblin"}) {
		t.Fatal("active long charge released attacker slot")
	}
}

func TestBossChargeHitsOnceAndRespectsAttackBudget(t *testing.T) {
	r := bossChargeRun()
	before := r.Player.HP
	for n := 0; n < 150 && r.Enemies[0].ChargeRecovery == 0; n++ {
		r.enemyTick(0, .02)
	}
	if r.Player.HP >= before {
		t.Fatal("charge did not hit grounded target")
	}
	hp := r.Player.HP
	for n := 0; n < 50; n++ {
		r.enemyTick(0, .02)
	}
	if r.Player.HP != hp {
		t.Fatal("charge hit again in recovery")
	}
	r = bossChargeRun()
	r.Level = &Level{Rooms: []Arena{{MaxAttackers: 1}}}
	r.Enemies = append(r.Enemies, Actor{ID: "busy", Kind: "goblin", HP: 100, Windup: .5})
	r.enemyTick(0, .02)
	if r.Enemies[0].Windup > 0 {
		t.Fatal("boss ignored attacker budget")
	}
}
