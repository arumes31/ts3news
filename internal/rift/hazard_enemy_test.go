package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func enemyRockRun() *Run {
	r := testRun()
	r.SkillTimers = map[string]float64{}
	r.Clock = 1.25
	r.Level = &Level{ID: 91, Region: 9, Rooms: []Arena{{Hazards: []Hazard{{Kind: "falling_rock", Obstacle: Obstacle{450, 350, 100, 100}, Period: 5, Duration: .18}}}}}
	r.Player.X, r.Player.Y = 100, 330
	r.Enemies = []Actor{{ID: "rock-target", Kind: "goblin", X: 500, Y: 410, HP: 100, MaxHP: 100, Armor: .2}}
	return r
}

func TestFallingRockDamagesEnemyOnceAcrossSave(t *testing.T) {
	r := enemyRockRun()
	r.hazardTick()
	if math.Abs(r.Enemies[0].HP-83.2) > 1e-9 {
		t.Fatalf("rock damage not armor mitigated: %.2f", r.Enemies[0].HP)
	}
	if r.Stats.DamageDealt != 0 || r.Stats.ComboScore != 0 || r.AttackChain != 0 {
		t.Fatal("hazard counted as player attack")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	saved.hazardTick()
	if saved.Enemies[0].HP != r.Enemies[0].HP {
		t.Fatal("saved impact damaged twice")
	}
	for step := 0; step < 250; step++ {
		for key, value := range saved.SkillTimers {
			saved.SkillTimers[key] = math.Max(0, value-.02)
		}
	}
	saved.Clock += 5
	saved.hazardTick()
	if math.Abs(saved.Enemies[0].HP-66.4) > 1e-9 {
		t.Fatal("next rock did not damage")
	}
}

func TestFallingRockEnemyContactRules(t *testing.T) {
	for _, mode := range []string{"warning", "expired", "disabled", "outside", "burrowed", "prop", "other-hazard", "airborne"} {
		t.Run(mode, func(t *testing.T) {
			r := enemyRockRun()
			switch mode {
			case "warning":
				r.Clock = 1.1
			case "expired":
				r.Clock = 1.5
			case "disabled":
				r.Level.Rooms[0].Hazards[0].Disabled = true
			case "outside":
				r.Enemies[0].X = 600
			case "burrowed":
				r.Enemies[0].Burrowed = true
			case "prop":
				r.Enemies[0].Kind = "generator"
			case "other-hazard":
				r.Level.Rooms[0].Hazards[0].Kind = "poison"
			case "airborne":
				r.Enemies[0].Jump = 1
			}
			r.hazardTick()
			if (r.Enemies[0].HP < 100) != (mode == "airborne") {
				t.Fatal("incorrect rock contact eligibility")
			}
		})
	}
}

func TestFallingRockKillDropsOnceWithoutCombo(t *testing.T) {
	r := enemyRockRun()
	r.Enemies[0].HP = 1
	r.hazardTick()
	r.hazardTick()
	if r.Enemies[0].HP != 0 || r.Stats.Kills != 1 || len(r.Drops) != 1 {
		t.Fatal("rock kill did not use one normal defeat/drop")
	}
	if r.Stats.ComboScore != 0 || r.Stats.LargestHit != 0 {
		t.Fatal("rock kill inflated player combat stats")
	}
}

func TestFallingRockBossPhaseAndShield(t *testing.T) {
	r := enemyRockRun()
	e := &r.Enemies[0]
	e.Kind, e.ArtKey, e.Armor, e.Phase, e.HP = "boss", "monster:test", 0, 1, 71
	r.hazardTick()
	if e.HP != 50 || e.Phase != 2 {
		t.Fatalf("rock did not preserve boss phase transition: %+v", e)
	}
	r = enemyRockRun()
	e = &r.Enemies[0]
	e.Kind, e.ArtKey, e.Armor, e.BossShieldHP, e.BossShieldMax = "boss", "monster:test", 0, 30, 30
	r.hazardTick()
	if e.HP != 100 || e.BossShieldHP != 9 {
		t.Fatal("rock bypassed boss shield")
	}
}

func TestFallingRockDoesNotBorrowPlayerBonuses(t *testing.T) {
	r := enemyRockRun()
	r.Build.Class = "berserker"
	r.Player.MaxHP = 100
	r.Player.HP = 30
	e := &r.Enemies[0]
	e.Shield, e.Facing, e.Summoned, e.ArrivalVulnerability = true, 1, true, 1
	r.hazardTick()
	if math.Abs(e.HP-83.2) > 1e-9 || r.Stats.RearStrikes != 0 || r.Stats.SummonPunishes != 0 || e.RecoilX != 0 {
		t.Fatal("rock borrowed player offense or directional recoil")
	}
	r = enemyRockRun()
	e = &r.Enemies[0]
	e.Kind, e.ArtKey, e.WeakPoint = "boss", "monster:test", 1
	r.hazardTick()
	if math.Abs(e.HP-83.2) > 1e-9 || e.BossStagger != 0 {
		t.Fatal("rock borrowed weak-point or stagger bonus")
	}
}

func TestFallingRockSharedCooldownPauseAndPractice(t *testing.T) {
	r := enemyRockRun()
	r.Level.Rooms[0].Hazards = append(r.Level.Rooms[0].Hazards, r.Level.Rooms[0].Hazards[0])
	r.Paused = true
	r.Step(Input{}, time.UnixMilli(r.LastMS+100))
	if r.Enemies[0].HP != 100 {
		t.Fatal("paused rock damaged enemy")
	}
	r.Paused = false
	r.hazardTick()
	if math.Abs(r.Enemies[0].HP-83.2) > 1e-9 {
		t.Fatal("overlapping rocks stacked damage")
	}
	r.spawnRoom()
	if r.SkillTimers["hazard-enemy-0-rock-target"] != 0 {
		t.Fatal("enemy cooldown leaked across rooms")
	}
	r = enemyRockRun()
	r.Practice = &PracticeState{Mode: "boss"}
	r.Enemies[0].HP = 1
	r.hazardTick()
	if r.Stats.Kills != 0 || len(r.Drops) != 0 || r.Practice.Hits != 0 {
		t.Fatal("practice rock produced rewards or player hits")
	}
}
