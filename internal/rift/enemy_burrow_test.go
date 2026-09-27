package rift

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

func burrowRun() *Run {
	r := testRun()
	r.Player.X, r.Player.Y = 500, 410
	r.Enemies = []Actor{{ID: "burrower", Kind: "goblin", Burrowing: true, X: 700, Y: 410, HP: 100, MaxHP: 100, Damage: 20, Speed: 90}}
	r.Level = &Level{Rooms: []Arena{{}}}
	return r
}

func TestBurrowLocksApproachAndWarnsBeforeSingleEmergence(t *testing.T) {
	r := burrowRun()
	hp := r.Player.HP
	r.enemyTick(0, .02)
	e := &r.Enemies[0]
	if !e.Burrowed || e.AttackName != "Burrow" || e.Windup < 1.8 || e.TargetX != 500 {
		t.Fatal("burrow did not start with locked target and full warning")
	}
	r.enemyTick(0, .84)
	if math.Abs(e.X-500) > .01 || e.Windup < .98 || r.Player.HP != hp {
		t.Fatal("approach must arrive before a stationary one-second warning")
	}
	r.enemyTick(0, .9)
	if r.Player.HP != hp {
		t.Fatal("emerged before warning expired")
	}
	r.enemyTick(0, .1)
	if e.Burrowed || e.Attacks != 1 || e.BurrowRecovery != .9 || r.Player.HP >= hp {
		t.Fatal("emergence must hit once then recover")
	}
	hp = r.Player.HP
	x := e.X
	r.enemyTick(0, .5)
	if r.Player.HP != hp || e.X != x || r.Stats.Kills != 0 || len(r.Drops) != 0 {
		t.Fatal("recovery moved, repeated damage or granted rewards")
	}
}

func TestBurrowTargetStaysFixedAcrossSaveAndPlayerEscape(t *testing.T) {
	for _, escape := range []string{"move", "jump", "dodge"} {
		r := burrowRun()
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
		switch escape {
		case "move":
			saved.Player.Y = 480
		case "jump":
			saved.Player.Jump = .4
		case "dodge":
			saved.SkillTimers["dodge_invulnerability"] = 1
		}
		for i := 0; i < 80; i++ {
			saved.enemyTick(0, .02)
		}
		if saved.Enemies[0].TargetX != 500 || saved.Enemies[0].TargetY != 410 || saved.Player.HP != saved.Player.MaxHP || saved.Enemies[0].Attacks != 1 {
			t.Fatalf("saved %s counterplay failed", escape)
		}
	}
}

func TestBurrowDamageAndControlInterruptBeforeEmergence(t *testing.T) {
	for _, mode := range []string{"damage", "stagger", "knockdown", "death"} {
		r := burrowRun()
		r.enemyTick(0, .02)
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
		if r.Enemies[0].Burrowed || r.Enemies[0].Windup != 0 || r.Enemies[0].Cooldown < 1.9 {
			t.Fatalf("%s did not interrupt", mode)
		}
		for i := 0; i < 50; i++ {
			r.enemyTick(0, .02)
		}
		if findEvent(r.Events, "burrow_emerge") != nil || r.Player.HP != r.Player.MaxHP {
			t.Fatalf("%s still emerged", mode)
		}
	}
}

func TestCanonicalRatsUseBurrowingApproach(t *testing.T) {
	noun := "Rat"
	if nouns := i18n.Pool("pool.mob.noun"); len(nouns) > 0 {
		noun = nouns[0]
	}
	count := 0
	for _, mob := range content.AbyssMobCatalog() {
		if mob.Type == content.MobCommon && strings.HasSuffix(mob.Name, " "+noun) {
			count++
			if !AdaptMonster(mob).Burrowing {
				t.Fatal("canonical rat lacks burrowing role")
			}
		}
	}
	if count == 0 {
		t.Fatal("no catalog burrowers exercised")
	}
}

func TestBurrowRespectsWallsBudgetAndCommittedActions(t *testing.T) {
	for _, late := range []bool{false, true} {
		r := burrowRun()
		if late {
			r.enemyTick(0, .02)
		}
		r.Level.Rooms[0].Obstacles = []Obstacle{{X: 580, Y: 350, W: 30, H: 130}}
		r.enemyTick(0, .84)
		if r.Enemies[0].Burrowed || r.Enemies[0].X < 610 || r.Player.HP != r.Player.MaxHP {
			t.Fatal("burrow crossed a wall or damaged through it")
		}
	}
	r := burrowRun()
	r.Level.Rooms[0].MaxAttackers = 1
	r.Enemies = append(r.Enemies, Actor{ID: "busy", Kind: "goblin", HP: 100, Windup: 1})
	r.enemyTick(0, .02)
	if r.Enemies[0].Burrowed {
		t.Fatal("burrow exceeded attacker limit")
	}
	r = burrowRun()
	r.Enemies[0].Pack = true
	r.PackAttackLockout = .4
	r.enemyTick(0, .02)
	if r.Enemies[0].Burrowed {
		t.Fatal("burrow ignored pack lockout")
	}
	r = burrowRun()
	r.Enemies[0].Windup = .01
	r.enemyTick(0, .02)
	if r.Enemies[0].Burrowed || r.Enemies[0].Attacks != 1 {
		t.Fatal("burrow replaced a committed attack")
	}
	for _, kind := range []string{"boss", "archer", "treasure", "totem"} {
		r = burrowRun()
		r.Enemies[0].Kind = kind
		r.enemyTick(0, .02)
		if r.Enemies[0].Burrowed {
			t.Fatalf("burrow replaced %s role", kind)
		}
	}
}

func TestBurrowRetainsRecoveryAndAllowsWalkingEscape(t *testing.T) {
	r := burrowRun()
	r.enemyTick(0, .02)
	for i := 0; i < 95; i++ {
		r.enemyTick(0, .02)
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
	saved.Player.X = 300
	saved.enemyTick(0, .3)
	if saved.Enemies[0].X != x || saved.Enemies[0].BurrowRecovery <= 0 {
		t.Fatal("saved recovery was lost")
	}
	r = burrowRun()
	r.enemyTick(0, .02)
	r.enemyTick(0, .84)
	r.enemyTick(0, .3)
	for i := 0; i < 35; i++ {
		r.moveActor(&r.Player, 0, 3, false)
		r.enemyTick(0, .02)
	}
	if r.Player.HP != r.Player.MaxHP || r.Enemies[0].Attacks != 1 {
		t.Fatal("walking escape after reaction delay failed")
	}
}

func TestBurrowCancelledAtAuthoredTerrainResumesWithWorldClock(t *testing.T) {
	r := NewRunAtLevel("pursuit-audit", Build{HP: 10000}, time.Unix(0, 0), content.AbyssMobCatalog(), 71)
	spawn := r.EncounterPlan[0][0]
	spawn.Patrol = false
	spawn.Alerted = true
	r.Enemies = []Actor{spawn}
	r.RoomObjective = nil
	r.Projectiles = nil
	r.PackAttackLockout = 0
	cancelled := false
	for step := 0; step < 3000; step++ {
		r.tick(Input{}, .02)
		cancelled = cancelled || findEvent(r.Events, "burrow_cancel") != nil
		if r.Enemies[0].Attacks > 0 {
			if !cancelled {
				t.Fatal("authored terrain did not exercise burrow cancellation")
			}
			t.Logf("Pack fighter resumed after terrain cancellation at %.2fs", float64(step+1)*.02)
			return
		}
	}
	t.Fatalf("world clock did not release pack fighter: %+v", r.Enemies[0])
}
