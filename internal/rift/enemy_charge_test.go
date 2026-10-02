package rift

import (
	"encoding/json"
	"math"
	"testing"
	"ts3news/internal/content"
)

func chargeRun() *Run {
	r := testRun()
	r.Player.X, r.Player.Y = 350, 410
	r.Enemies = []Actor{{ID: "charger", Kind: "goblin", Charging: true, X: 150, Y: 410, HP: 100, MaxHP: 100, Damage: 20, Speed: 90}}
	return r
}

func TestEnemyChargeLocksAimAndLeavesSavedRecovery(t *testing.T) {
	r := chargeRun()
	r.enemyTick(0, .02)
	e := &r.Enemies[0]
	if e.AttackName != "Charge" || e.Windup < .6 {
		t.Fatal("missing charge warning")
	}
	targetX, targetY := e.TargetX, e.TargetY
	r.Player.Y = 480
	for n := 0; n < 100 && e.ChargeRecovery == 0; n++ {
		r.enemyTick(0, .02)
	}
	if e.ChargeRecovery == 0 || e.ChargeActive || e.TargetX != targetX || e.TargetY != targetY || r.Player.HP != r.Player.MaxHP {
		t.Fatalf("no safe dodge/recovery: %+v", e)
	}
	x, y, hp := e.X, e.Y, e.HP
	r.hurtEnemy(0, 10, "hit")
	if e.HP >= hp {
		t.Fatal("recovery could not be punished")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 30; n++ {
		saved.enemyTick(0, .02)
	}
	e = &saved.Enemies[0]
	if e.ChargeRecovery <= 0 || e.X != x || e.Y != y || e.Windup != 0 {
		t.Fatal("recovery moved, attacked, or vanished on reload")
	}
	for n := 0; n < 100; n++ {
		saved.enemyTick(0, .02)
	}
	if e.ChargeRecovery > 0 || math.Hypot(e.X-x, e.Y-y) < 1 {
		t.Fatal("enemy never resumed pursuit")
	}
}

func TestEnemyChargeHitsOnceAndStopsAtCover(t *testing.T) {
	r := chargeRun()
	before := r.Player.HP
	for n := 0; n < 100 && r.Enemies[0].ChargeRecovery == 0; n++ {
		r.enemyTick(0, .02)
	}
	if r.Player.HP >= before {
		t.Fatal("charge never connected")
	}
	hp := r.Player.HP
	for n := 0; n < 30; n++ {
		r.enemyTick(0, .02)
	}
	if r.Player.HP != hp {
		t.Fatal("charge damaged repeatedly in recovery")
	}
	r = chargeRun()
	r.Enemies[0].X = 700
	r.Player.X = 900
	r.Player.Y = 390
	r.Enemies[0].Y = 390
	r.Level = &Level{Rooms: []Arena{{HighCover: []Obstacle{{780, 350, 40, 100}}}}}
	r.enemyTick(0, .02)
	for n := 0; n < 100 && r.Enemies[0].ChargeRecovery == 0; n++ {
		r.enemyTick(0, .02)
	}
	if r.Enemies[0].ChargeRecovery == 0 || r.Enemies[0].X >= 780 {
		t.Fatalf("charge crossed cover: %+v", r.Enemies[0])
	}
}

func TestEnemyChargeCanBeInterruptedBeforeOrDuringRush(t *testing.T) {
	for _, active := range []bool{false, true} {
		r := chargeRun()
		r.enemyTick(0, .02)
		if active {
			for n := 0; n < 40 && !r.Enemies[0].ChargeActive; n++ {
				r.enemyTick(0, .02)
			}
		}
		r.hurtEnemy(0, 10, "hit")
		e := &r.Enemies[0]
		if e.ChargeActive || e.Windup != 0 || e.ChargeRecovery <= 0 {
			t.Fatal("hit did not cancel charge")
		}
		x := e.X
		r.enemyTick(0, .02)
		if e.X != x {
			t.Fatal("cancelled charge resumed")
		}
	}
}

func TestCanonicalCatalogIncludesChargingFighters(t *testing.T) {
	count := 0
	for _, mob := range content.AbyssMobCatalog() {
		a := AdaptMonster(mob)
		if a.Charging && a.Kind != "boss" && a.Kind != "archer" && a.Kind != "treasure" {
			count++
			t.Log(a.Name)
		}
	}
	if count == 0 {
		t.Fatal("charge behavior is not used by any canonical fighter")
	}
}

func TestEnemyChargeRespectsCrowdingAndControlInterrupts(t *testing.T) {
	r := chargeRun()
	r.Enemies[0].Pack = true
	r.PackAttackLockout = .4
	r.enemyTick(0, .02)
	if r.Enemies[0].AttackName == "Charge" {
		t.Fatal("charge ignored pack spacing")
	}
	for _, kind := range []string{"archer", "boss", "treasure"} {
		r = chargeRun()
		r.Enemies[0].Kind = kind
		r.enemyTick(0, .02)
		if r.Enemies[0].AttackName == "Charge" {
			t.Fatal("charge replaced specialized role")
		}
	}
	for _, active := range []bool{false, true} {
		r = chargeRun()
		r.enemyTick(0, .02)
		if active {
			for n := 0; n < 40 && !r.Enemies[0].ChargeActive; n++ {
				r.enemyTick(0, .02)
			}
		}
		r.Enemies[0].Knockdown = .3
		r.Enemies[0].Windup = 0
		r.enemyTick(0, .02)
		if r.Enemies[0].ChargeActive || r.Enemies[0].AttackName == "Charge" || r.Enemies[0].ChargeRecovery == 0 {
			t.Fatal("control effect did not cancel charge")
		}
	}
}

func TestEnemyChargeSaveDuringWarningAndRushAndJumpEscape(t *testing.T) {
	for _, active := range []bool{false, true} {
		r := chargeRun()
		r.enemyTick(0, .02)
		if active {
			for n := 0; n < 40 && !r.Enemies[0].ChargeActive; n++ {
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
		before := saved.Player.HP
		for n := 0; n < 100 && saved.Enemies[0].ChargeRecovery == 0; n++ {
			saved.enemyTick(0, .02)
		}
		if saved.Enemies[0].ChargeRecovery == 0 || saved.Player.HP != before {
			t.Fatal("saved charge failed to finish or hit airborne player")
		}
	}
	r := chargeRun()
	r.Level = &Level{Rooms: []Arena{{MaxAttackers: 1}}}
	r.Enemies = append(r.Enemies, Actor{ID: "busy", Kind: "goblin", HP: 100, Windup: .5})
	r.enemyTick(0, .02)
	if r.Enemies[0].AttackName == "Charge" {
		t.Fatal("charge exceeded arena attacker limit")
	}
}
