package rift

import (
	"encoding/json"
	"testing"
	"ts3news/internal/content"
)

func armingRun() *Run {
	r := testRun()
	r.Player.X, r.Player.Y = 500, 410
	r.Enemies = []Actor{{ID: "volatile", Kind: "goblin", Explosive: true, X: 580, Y: 410, HP: 100, MaxHP: 100, Damage: 20, Speed: 90}}
	r.Level = &Level{Rooms: []Arena{{}}}
	return r
}

func TestExplosiveEnemyArmsThenBlastsOnce(t *testing.T) {
	r := armingRun()
	hp := r.Player.HP
	r.enemyTick(0, .02)
	if r.Enemies[0].ArmingTimer != 1.1 || r.Enemies[0].Windup != 1.1 || r.Enemies[0].AttackName != "Blast" {
		t.Fatal("missing arming phase")
	}
	x := r.Enemies[0].X
	r.enemyTick(0, 1)
	if r.Player.HP != hp || r.Enemies[0].X != x {
		t.Fatal("arming moved or hit early")
	}
	r.enemyTick(0, .11)
	if r.Player.HP >= hp || r.Enemies[0].ArmingTimer != 0 || r.Enemies[0].HP != 100 || r.Enemies[0].Cooldown != 2 {
		t.Fatal("blast did not resolve once with recovery")
	}
	hp = r.Player.HP
	r.enemyTick(0, .1)
	if r.Player.HP != hp || r.Stats.Kills != 0 || len(r.Drops) != 0 {
		t.Fatal("blast repeated or fabricated rewards")
	}
}

func TestArmingInterruptionsCancelAndDelayRetry(t *testing.T) {
	for _, mode := range []string{"damage", "knockdown", "stagger", "death"} {
		r := armingRun()
		r.enemyTick(0, .02)
		switch mode {
		case "damage":
			r.hurtEnemy(0, 1, "hit")
		case "death":
			r.hurtEnemy(0, 1000, "hit")
		case "knockdown":
			r.Enemies[0].Knockdown = .3
			r.enemyTick(0, .02)
		case "stagger":
			r.Enemies[0].Pose = "stagger"
			r.Enemies[0].PoseTime = .3
			r.enemyTick(0, .02)
		}
		e := r.Enemies[0]
		if e.ArmingTimer != 0 || e.Windup != 0 || e.AttackName != "" || e.Cooldown < 1.3 {
			t.Fatalf("failed %s interruption", mode)
		}
		for n := 0; n < 40; n++ {
			r.enemyTick(0, .02)
		}
		if findEvent(r.Events, "enemy_blast") != nil || r.Player.HP != r.Player.MaxHP {
			t.Fatal("cancelled arming still exploded")
		}
	}
}

func TestSavedArmingAllowsEscapeAndCover(t *testing.T) {
	for _, mode := range []string{"move", "jump", "dodge", "wall"} {
		r := armingRun()
		r.enemyTick(0, .02)
		r.enemyTick(0, .4)
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved Run
		if err = json.Unmarshal(raw, &saved); err != nil {
			t.Fatal(err)
		}
		saved.Events = nil
		switch mode {
		case "move":
			saved.Player.Y = 490
		case "jump":
			saved.Player.Jump = .5
		case "dodge":
			saved.SkillTimers["dodge_invulnerability"] = 1
		case "wall":
			saved.Level.Rooms[0].HighCover = []Obstacle{{540, 350, 10, 100}}
		}
		saved.enemyTick(0, .71)
		if saved.Player.HP != saved.Player.MaxHP || findEvent(saved.Events, "enemy_blast") == nil || findEvent(saved.Events, "arming_start") != nil {
			t.Fatalf("failed saved %s escape or cue continuity", mode)
		}
	}
}

func TestCanonicalFireEnemiesHaveExplosiveRole(t *testing.T) {
	count := 0
	for _, mob := range content.AbyssMobCatalog() {
		a := AdaptMonster(mob)
		if mob.Type == content.MobCommon && a.Shot == "fire" {
			count++
			if !a.Explosive {
				t.Fatal("canonical fire enemy has no arming role")
			}
		}
	}
	if count == 0 {
		t.Fatal("arming behavior unused by catalog")
	}
}

func TestArmingPreservesCommittedAttacksAndBudget(t *testing.T) {
	r := armingRun()
	r.Enemies[0].Windup = .01
	r.enemyTick(0, .02)
	if r.Enemies[0].ArmingTimer != 0 || r.Enemies[0].Attacks != 1 {
		t.Fatal("arming replaced committed attack")
	}
	r = armingRun()
	r.Level.Rooms[0].MaxAttackers = 1
	r.Enemies = append(r.Enemies, Actor{ID: "busy", Kind: "goblin", HP: 100, Windup: .5})
	r.enemyTick(0, .02)
	if r.Enemies[0].ArmingTimer != 0 {
		t.Fatal("arming ignored attacker budget")
	}
	r = armingRun()
	r.Enemies[0].Pack = true
	r.PackAttackLockout = .4
	r.enemyTick(0, .02)
	if r.Enemies[0].ArmingTimer != 0 {
		t.Fatal("arming ignored pack lockout")
	}
	for _, kind := range []string{"boss", "treasure", "totem"} {
		r = armingRun()
		r.Enemies[0].Kind = kind
		r.enemyTick(0, .02)
		if r.Enemies[0].ArmingTimer != 0 {
			t.Fatalf("arming replaced %s role", kind)
		}
	}
}
