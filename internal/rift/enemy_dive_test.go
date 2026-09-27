package rift

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

func diveRun() *Run {
	r := testRun()
	r.Player.X, r.Player.Y = 500, 410
	r.Enemies = []Actor{{ID: "bat", Kind: "goblin", Flying: true, X: 700, Y: 410, HP: 100, MaxHP: 100, Damage: 20, Speed: 90}}
	r.Level = &Level{Rooms: []Arena{{}}}
	return r
}
func TestDiveWarnsBeforeLockedSwoopAndRecovers(t *testing.T) {
	r := diveRun()
	hp := r.Player.HP
	r.enemyTick(0, .02)
	e := &r.Enemies[0]
	if e.AttackName != "Dive" || e.Windup != 1 || e.Diving || e.TargetX != 500 {
		t.Fatal("missing fixed dive warning")
	}
	r.enemyTick(0, .98)
	if e.X != 700 || r.Player.HP != hp || e.Diving {
		t.Fatal("dive moved or hit before warning")
	}
	r.enemyTick(0, .02)
	if !e.Diving || e.Attacks != 1 {
		t.Fatal("warning did not release one dive")
	}
	for i := 0; i < 40 && e.Diving; i++ {
		r.enemyTick(0, .02)
	}
	if e.Diving || e.DiveRecovery != 1 || r.Player.HP >= hp {
		t.Fatal("dive did not hit once and land")
	}
	hp = r.Player.HP
	x := e.X
	r.enemyTick(0, .4)
	if e.X != x || r.Player.HP != hp || r.Stats.Kills != 0 || len(r.Drops) != 0 {
		t.Fatal("recovery moved or fabricated hits/rewards")
	}
}
func TestSavedDiveAllowsMovementJumpAndDodge(t *testing.T) {
	for _, mode := range []string{"move", "jump", "dodge"} {
		r := diveRun()
		r.enemyTick(0, .02)
		r.enemyTick(0, .4)
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var s Run
		if err = json.Unmarshal(raw, &s); err != nil {
			t.Fatal(err)
		}
		switch mode {
		case "move":
			s.Player.Y = 480
		case "jump":
			s.Player.Jump = .4
		case "dodge":
			s.SkillTimers["dodge_invulnerability"] = 1
		}
		for i := 0; i < 75; i++ {
			s.enemyTick(0, .02)
		}
		if s.Player.HP != s.Player.MaxHP || s.Enemies[0].Attacks != 1 || s.Enemies[0].TargetY != 410 || s.Enemies[0].Diving {
			t.Fatalf("saved %s counterplay failed", mode)
		}
	}
}
func TestDiveInterruptionsAndTerrain(t *testing.T) {
	for _, active := range []bool{false, true} {
		for _, mode := range []string{"damage", "stagger", "knockdown", "death"} {
			r := diveRun()
			r.enemyTick(0, .02)
			if active {
				r.enemyTick(0, 1)
			}
			switch mode {
			case "damage":
				r.hurtEnemy(0, 1, "hit")
			case "death":
				r.hurtEnemy(0, 1000, "hit")
			case "stagger":
				r.Enemies[0].Pose = "stagger"
				r.Enemies[0].PoseTime = .3
			case "knockdown":
				r.Enemies[0].Knockdown = .3
			}
			r.enemyTick(0, .02)
			if r.Enemies[0].Diving || r.Enemies[0].Windup != 0 || r.Enemies[0].AttackName != "" {
				t.Fatalf("%s active=%v did not cancel", mode, active)
			}
			for i := 0; i < 50; i++ {
				r.enemyTick(0, .02)
			}
			if findEvent(r.Events, "dive_land") != nil {
				t.Fatal("interruption emitted a successful landing cue")
			}
			if r.Player.HP != r.Player.MaxHP {
				t.Fatal("cancelled dive still hit")
			}
		}
	}
	r := diveRun()
	r.enemyTick(0, .02)
	r.Level.Rooms[0].Obstacles = []Obstacle{{X: 580, Y: 350, W: 30, H: 130}}
	r.enemyTick(0, 1)
	for i := 0; i < 40 && r.Enemies[0].Diving; i++ {
		r.enemyTick(0, .02)
	}
	if r.Enemies[0].X < 610 || r.Enemies[0].Diving || r.Player.HP != r.Player.MaxHP {
		t.Fatal("dive crossed wall")
	}
}
func TestDiveBudgetCommitmentAndCatalog(t *testing.T) {
	r := diveRun()
	r.Level.Rooms[0].MaxAttackers = 1
	r.Enemies = append(r.Enemies, Actor{Kind: "goblin", HP: 100, Windup: .5})
	r.enemyTick(0, .02)
	if r.Enemies[0].AttackName == "Dive" {
		t.Fatal("attack budget ignored")
	}
	r = diveRun()
	r.Enemies[0].Pack = true
	r.PackAttackLockout = .4
	r.enemyTick(0, .02)
	if r.Enemies[0].AttackName == "Dive" {
		t.Fatal("pack spacing ignored")
	}
	r = diveRun()
	r.Enemies[0].Windup = .01
	r.enemyTick(0, .02)
	if r.Enemies[0].AttackName == "Dive" || r.Enemies[0].Attacks != 1 {
		t.Fatal("committed attack replaced")
	}
	for _, kind := range []string{"boss", "archer", "treasure", "totem"} {
		r = diveRun()
		r.Enemies[0].Kind = kind
		r.enemyTick(0, .02)
		if r.Enemies[0].AttackName == "Dive" {
			t.Fatal("special role replaced")
		}
	}
	noun := "Bat"
	if nouns := i18n.Pool("pool.mob.noun"); len(nouns) > 7 {
		noun = nouns[7]
	}
	count := 0
	for _, m := range content.AbyssMobCatalog() {
		if m.Type == content.MobCommon && strings.HasSuffix(m.Name, " "+noun) {
			count++
			if !AdaptMonster(m).Flying {
				t.Fatal("canonical bat has no dive role")
			}
		}
	}
	if count == 0 {
		t.Fatal("no canonical bats exercised")
	}
}
func TestDiveRecoveryPersistsAndWalkingEscapeIsPossible(t *testing.T) {
	r := diveRun()
	r.enemyTick(0, .02)
	r.enemyTick(0, 1)
	r.Player.Y = 480
	for i := 0; i < 40 && r.Enemies[0].Diving; i++ {
		r.enemyTick(0, .02)
	}
	if math.Abs(r.Enemies[0].X-500) > 1 {
		t.Fatal("missed dive did not finish at locked point")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var s Run
	if err = json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	x := s.Enemies[0].X
	s.enemyTick(0, .3)
	if s.Enemies[0].X != x || s.Enemies[0].DiveRecovery <= 0 {
		t.Fatal("saved recovery lost")
	}
	r = diveRun()
	r.enemyTick(0, .02)
	r.enemyTick(0, .3)
	for i := 0; i < 75; i++ {
		r.moveActor(&r.Player, 0, 3, false)
		r.enemyTick(0, .02)
	}
	if r.Player.HP != r.Player.MaxHP || r.Enemies[0].Attacks != 1 {
		t.Fatal("walking counterplay failed")
	}
}

func TestActiveDiveSaveKeepsAimAndAttackSlot(t *testing.T) {
	r := diveRun()
	r.Enemies[0].X = 800
	r.Level.Rooms[0].MaxAttackers = 1
	r.enemyTick(0, .02)
	r.enemyTick(0, 1)
	r.enemyTick(0, .4)
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var s Run
	if err = json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	s.Player.Y = 480
	s.enemyTick(0, .5)
	candidate := Actor{ID: "other", Kind: "goblin", HP: 100}
	if !s.Enemies[0].Diving || s.Enemies[0].TargetY != 410 || s.canStartEnemyAttack(&candidate) {
		t.Fatal("saved dive lost its aim or attack slot")
	}
	for i := 0; i < 10; i++ {
		s.enemyTick(0, .02)
	}
	if s.Enemies[0].Diving || s.Player.HP != s.Player.MaxHP || s.Enemies[0].Attacks != 1 {
		t.Fatal("saved dive did not land once")
	}
}
