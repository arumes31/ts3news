package rift

import (
	"encoding/json"
	"testing"
)

func guardBreakRun() *Run {
	r := testRun()
	r.Enemies = []Actor{{ID: "boss", Kind: "boss", ArtKey: "monster:test", HP: 1000, MaxHP: 1000, Armor: .4, X: 500, Y: 410, Phase: 1}}
	return r
}

func TestBossStaggerThresholdIsSeparateFromHealth(t *testing.T) {
	for _, damage := range []float64{1, 20} {
		r := guardBreakRun()
		for n := 1; n <= 4; n++ {
			r.hurtEnemy(0, damage, "hit")
			if r.Enemies[0].BossStagger != float64(n*20) || r.Enemies[0].BossGuardBreak != 0 {
				t.Fatal("wrong independent stagger gain")
			}
		}
		r.Enemies[0].Windup = .5
		r.Enemies[0].AttackName = "Slam"
		r.hurtEnemy(0, damage, "hit")
		e := r.Enemies[0]
		if e.BossStagger != 0 || e.BossGuardBreak != 1.5 || e.Windup != 0 || e.AttackName != "" || e.Pose != "stagger" || findEvent(r.Events, "boss_guard_break") == nil {
			t.Fatalf("missing threshold break: %+v", e)
		}
	}
}

func TestBossGuardBreakArmorRecoveryAndGraceSurviveSave(t *testing.T) {
	r := guardBreakRun()
	for n := 0; n < 5; n++ {
		r.hurtEnemy(0, 1, "hit")
	}
	e := &r.Enemies[0]
	before := e.HP
	r.hurtEnemy(0, 10, "hit")
	if before-e.HP != 8 {
		t.Fatalf("broken armor: dealt %v", before-e.HP)
	}
	if e.BossGuardBreak != 1.5 || e.BossStagger != 0 {
		t.Fatal("hit extended break or filled meter")
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
	x, y := e.X, e.Y
	saved.enemyTick(0, 1)
	e = &saved.Enemies[0]
	if e.X != x || e.Y != y || e.BossGuardBreak != .5 || e.Windup != 0 {
		t.Fatal("saved break moved or attacked")
	}
	saved.enemyTick(0, .5)
	if e.BossGuardBreak != 0 || e.BossStaggerGrace != 2 {
		t.Fatal("break did not recover with grace")
	}
	before = e.HP
	saved.hurtEnemy(0, 10, "hit")
	if before-e.HP != 6 || e.BossStagger != 0 {
		t.Fatal("armor or immunity failed to recover")
	}
	saved.enemyTick(0, 2)
	saved.hurtEnemy(0, 1, "hit")
	if e.BossStagger != 20 {
		t.Fatal("meter never became available again")
	}
	if findEvent(saved.Events, "boss_guard_break") != nil {
		t.Fatal("save replayed break cue")
	}
}

func TestBossStaggerExcludesZeroDamageDeathAndOtherRoles(t *testing.T) {
	r := guardBreakRun()
	r.hurtEnemy(0, 0, "hit")
	if r.Enemies[0].BossStagger != 0 {
		t.Fatal("zero damage stagger")
	}
	r.Enemies[0].BossStagger = 80
	r.hurtEnemy(0, 2000, "hit")
	if r.Enemies[0].BossGuardBreak != 0 || findEvent(r.Events, "boss_guard_break") != nil {
		t.Fatal("dead boss entered break")
	}
	r = guardBreakRun()
	r.Enemies[0].Kind = "knight"
	for n := 0; n < 6; n++ {
		r.hurtEnemy(0, 1, "hit")
	}
	if r.Enemies[0].BossStagger != 0 || r.Enemies[0].BossGuardBreak != 0 {
		t.Fatal("ordinary enemy got boss meter")
	}
}

func TestBossBreakReplacesChargeRecoveryAndSurvivesPhaseChange(t *testing.T) {
	r := guardBreakRun()
	e := &r.Enemies[0]
	e.BossStagger = 80
	e.Charging = true
	e.ChargeActive = true
	e.AttackName = "Charge"
	r.hurtEnemy(0, 1, "hit")
	if e.ChargeActive || e.ChargeRecovery > 0 || e.BossGuardBreak != 1.5 {
		t.Fatal("charge recovery leaked into guard break")
	}
	r.hurtEnemy(0, 900, "hit")
	r.enemyTick(0, .1)
	if e.Phase != 2 || e.BossGuardBreak != 1.4 || e.PoseTime != 1.4 || e.Windup > 0 {
		t.Fatalf("phase shortened break: %+v", e)
	}
}

func TestPartialBossMeterSurvivesSaveWithoutCue(t *testing.T) {
	r := guardBreakRun()
	for n := 0; n < 4; n++ {
		r.hurtEnemy(0, 1, "hit")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Enemies[0].BossStagger != 80 {
		t.Fatal("partial meter lost")
	}
	saved.Events = nil
	saved.hurtEnemy(0, 1, "hit")
	if saved.Enemies[0].BossGuardBreak != 1.5 || findEvent(saved.Events, "boss_guard_break") == nil {
		t.Fatal("saved meter failed to break")
	}
}

func TestBossResumesWithWarningAfterGuardBreak(t *testing.T) {
	r := guardBreakRun()
	r.Player.X = 450
	for n := 0; n < 5; n++ {
		r.hurtEnemy(0, 1, "hit")
	}
	for n := 0; n < 110; n++ {
		r.enemyTick(0, .02)
	}
	e := r.Enemies[0]
	if e.BossGuardBreak != 0 || e.Windup <= 0 || e.Attacks != 0 {
		t.Fatal("boss failed to resume with telegraph after recovery")
	}
}
