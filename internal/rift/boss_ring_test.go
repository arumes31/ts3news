package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

func ringRun() *Run {
	r := testRun()
	r.Player.X, r.Player.Y = 500, 410
	e := AdaptMonster(content.Mob{Name: i18n.T("mob.void_lord"), Type: content.MobLegendary})
	e.ID, e.X, e.Y, e.Attacks = "void-lord", 650, 410, 2
	r.Enemies = []Actor{e}
	r.Level = &Level{Rooms: []Arena{{}}}
	return r
}
func TestVoidRingSequenceWarningAndSavedImpact(t *testing.T) {
	r := ringRun()
	e := &r.Enemies[0]
	plan := r.NextBossAttack(*e)
	if !e.RingAttack || plan.Kind != "ring" || plan.Windup != 2.0 {
		t.Fatal("canonical Void Lord has no delayed ring")
	}
	r.enemyTick(0, .02)
	if e.TargetX != 650 || e.TargetY != 410 || e.RingGap != 0 || e.Windup != 2.0 {
		t.Fatal("ring center/gap not locked")
	}
	r.enemyTick(0, 1.9)
	if r.Player.HP != r.Player.MaxHP {
		t.Fatal("ring fired early")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var s Run
	if err = json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	s.Enemies[0].X = 800
	s.enemyTick(0, .11)
	if s.Player.HP >= s.Player.MaxHP || s.Enemies[0].Attacks != 3 || s.Enemies[0].Cooldown != 2.3 || findEvent(s.Events, "boss_ring") == nil {
		t.Fatal("saved ring did not resolve at original center")
	}
	for attack, kind := range []string{"slam", "projectile", "ring", "projectile", "slam", "projectile", "ring"} {
		e.Attacks = attack
		if r.NextBossAttack(*e).Kind != kind {
			t.Fatalf("sequence changed at %d", attack)
		}
	}
}
func TestRingSafeCenterExteriorGapAndCounterplay(t *testing.T) {
	for _, mode := range []string{"center", "outside", "gap", "jump", "dodge", "guard-break"} {
		r := ringRun()
		r.enemyTick(0, .02)
		switch mode {
		case "center":
			r.Player.X = 650
		case "outside":
			r.Player.X = 420
		case "gap":
			r.Player.X, r.Player.Y = 650, 340
		case "jump":
			r.Player.Jump = .4
		case "dodge":
			r.SkillTimers["dodge_invulnerability"] = 1
		case "guard-break":
			r.Enemies[0].BossStagger = 80
			r.hurtEnemy(0, 1, "hit")
		}
		r.enemyTick(0, 2.0)
		if r.Player.HP != r.Player.MaxHP {
			t.Fatalf("%s failed", mode)
		}
		if mode == "guard-break" && findEvent(r.Events, "boss_ring") != nil {
			t.Fatal("cancelled ring fired")
		}
	}
	r := ringRun()
	r.Practice = &PracticeState{SlowTelegraphs: true}
	r.enemyTick(0, .02)
	if r.Enemies[0].Windup != 4.0 {
		t.Fatal("slow practice lost warning")
	}
	r = ringRun()
	r.Player.Y = 350
	r.Enemies[0].Y = 350
	r.enemyTick(0, .02)
	if r.Enemies[0].RingGap != 1 {
		t.Fatal("gap did not face into arena")
	}
}
func TestRingDangerBoundaries(t *testing.T) {
	e := Actor{TargetX: 650, TargetY: 410, RingGap: 0}
	for _, c := range []struct {
		x, y   float64
		danger bool
	}{{750, 410, false}, {750.01, 410, true}, {850, 410, true}, {850.01, 410, false}, {650, 340, false}, {650, 480, true}, {725, 376.25, false}, {725.01, 376.25, true}} {
		if got := ringDanger(e, c.x, c.y); got != c.danger {
			t.Fatalf("boundary %.2f,%.2f got %v", c.x, c.y, got)
		}
	}
}

func TestRingHasWalkingEscapeInEveryFinalArena(t *testing.T) {
	checked := 0
	for _, level := range Campaign() {
		for phase := 1; phase <= 3; phase++ {
			r := NewRunAtLevel("ring-escape", Build{HP: 10000}, time.Unix(0, 0), content.AbyssMobCatalog(), level.ID)
			r.Room = 2
			r.spawnRoom()
			spawn := r.Enemies[0]
			boss := AdaptMonster(content.Mob{Name: i18n.T("mob.void_lord"), Type: content.MobLegendary})
			boss.ID, boss.X, boss.Y, boss.Phase, boss.Attacks = "ring-boss", spawn.X, spawn.Y, phase, 2
			r.Enemies = []Actor{boss}
			r.RoomObjective = nil
			for step := 0; step < 3000 && r.Enemies[0].Windup == 0; step++ {
				r.enemyTick(0, .02)
			}
			if r.Enemies[0].AttackName != "Void Ring" || r.Enemies[0].Windup != 2.0 || !ringDanger(r.Enemies[0], r.Player.X, r.Player.Y) {
				t.Fatalf("mission%d phase%d has no real ring threat", level.ID, phase)
			}
			raw, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var standing Run
			if err = json.Unmarshal(raw, &standing); err != nil {
				t.Fatal(err)
			}
			standing.enemyTick(0, 2.0)
			if standing.Player.HP >= r.Player.HP {
				t.Fatal("stationary ring control never hit")
			}
			escaped := false
			for _, direction := range [][2]float64{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
				var moving Run
				if err = json.Unmarshal(raw, &moving); err != nil {
					t.Fatal(err)
				}
				moving.enemyTick(0, .3)
				scale := 150 * .02 / math.Hypot(direction[0], direction[1])
				for step := 0; step < 90; step++ {
					moving.moveActor(&moving.Player, direction[0]*scale, direction[1]*scale, false)
					moving.enemyTick(0, .02)
				}
				if moving.Player.HP == r.Player.HP && moving.Enemies[0].Attacks == 3 {
					escaped = true
					break
				}
			}
			if !escaped {
				t.Fatalf("mission%d phase%d has no walking ring escape after300ms reaction", level.ID, phase)
			}
			checked++
		}
	}
	t.Logf("Checked%d real final-arena/phase ring escapes", checked)
}

// Regressions from the terrain grid: slow movement around bastion pillars must
// reach protected ground after a 300ms reaction without jumping or dodging.
func TestRingSlowedPillarShelterWithCombatTicks(t *testing.T) {
	for _, c := range []struct {
		mission int
		x, y    float64
	}{{3, 610, 402.5}, {13, 610, 402.5}, {23, 610, 402.5}, {33, 610, 402.5}, {43, 610, 402.5}, {33, 650, 442.5}, {43, 650, 442.5}} {
		r := ringRun()
		level := Campaign()[c.mission-1]
		r.Level, r.Room = &level, 2
		r.Player.X, r.Player.Y = c.x, c.y
		r.Player.HP, r.Player.MaxHP = 10000, 10000
		e := &r.Enemies[0]
		e.X, e.Y, e.TargetX, e.TargetY, e.RingGap = 800, 402.5, 800, 402.5, 0
		e.Windup, e.AttackName = r.NextBossAttack(*e).Windup, "Void Ring"
		r.SkillTimers["slowed"] = 5
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var moving Run
		if err := json.Unmarshal(raw, &moving); err != nil {
			t.Fatal(err)
		}
		turn := 43
		if c.y == 442.5 {
			turn = 67
		}
		sheltered, escaped := false, false
		for step := 0; step < 120; step++ {
			in := Input{}
			if step >= 15 && !sheltered {
				in.Y = -1
				if step-15 >= turn {
					in = Input{X: 1}
				}
			}
			beforeHP := moving.Player.HP
			moving.tick(in, .02)
			sheltered = moving.reservedBossArea(moving.Player.X, moving.Player.Y)
			if moving.Enemies[0].Attacks == 3 {
				escaped = sheltered && moving.Player.HP == beforeHP && moving.SkillTimers["slowed"] > 0
				break
			}
		}
		if !escaped {
			t.Errorf("mission %d start %.1f,%.1f has no tested slowed protected route", c.mission, c.x, c.y)
		}
	}
}
