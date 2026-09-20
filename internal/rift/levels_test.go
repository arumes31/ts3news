package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"ts3news/internal/content"
)

func TestCampaignEncounterPreviewsMatchFrozenCombat(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	templates := map[string]Actor{}
	for _, mob := range catalog {
		templates[mob.Name] = AdaptMonster(mob)
	}
	for _, level := range Campaign() {
		run := NewRunAtLevel("preview-test", Build{HP: 100}, time.Now(), catalog, level.ID)
		for room, arena := range level.Rooms {
			preview := arena.Encounter
			if preview == nil || preview.Enemies != len(run.EncounterPlan[room]) {
				t.Fatalf("mission %d tier %d count mismatch", level.ID, room+1)
			}
			for _, actor := range run.EncounterPlan[room] {
				base := templates[actor.Name]
				if math.Abs(actor.MaxHP-base.MaxHP*preview.HealthMultiplier) > 1e-8 || math.Abs(actor.Damage-base.Damage*preview.DamageMultiplier) > 1e-8 {
					t.Fatalf("mission %d tier %d scaling mismatch for %s", level.ID, room+1, actor.Name)
				}
			}
		}
	}
}

func TestCampaignHas100DistinctPlayableMissions(t *testing.T) {
	levels := Campaign()
	if len(levels) != 100 {
		t.Fatalf("got %d missions", len(levels))
	}
	names, layouts := map[string]bool{}, map[string]bool{}
	for index, level := range levels {
		if level.ID != index+1 || names[level.Name] || len(level.Rooms) != 3 {
			t.Fatalf("invalid mission %+v", level)
		}
		names[level.Name] = true
		geometry, _ := json.Marshal(level.Rooms[0].Obstacles)
		if layouts[string(geometry)] {
			t.Fatalf("mission %d repeats geometry", level.ID)
		}
		layouts[string(geometry)] = true
		r := NewRunAtLevel("reachability", Build{HP: 100}, time.Now(), content.AbyssMobCatalog(), level.ID)
		for room, arena := range level.Rooms {
			for _, h := range arena.Hazards {
				if h.Period <= 1.2+h.Duration || h.W <= 0 || h.Y < 315 || h.Y+h.H > 490 {
					t.Fatalf("invalid hazard %+v", h)
				}
			}
			// Flood the actual collision footprint: every enemy must be reachable
			// on foot, even without jumping or an equipped movement skill.
			type cell struct{ x, y int }
			start := cell{16, 41}
			queue := []cell{start}
			seen := map[cell]bool{start: true}
			for head := 0; head < len(queue); head++ {
				at := queue[head]
				for _, step := range []cell{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					next := cell{at.x + step.x, at.y + step.y}
					if seen[next] || next.x < 4 || next.x > 156 || next.y < 32 || next.y > 49 {
						continue
					}
					blocked := false
					for _, o := range arena.Obstacles {
						blocked = blocked || contains(o, float64(next.x*10), float64(next.y*10), 10)
					}
					if !blocked {
						seen[next] = true
						queue = append(queue, next)
					}
				}
			}
			for _, enemy := range r.EncounterPlan[room] {
				reached := false
				for at := range seen {
					if math.Hypot(float64(at.x*10)-enemy.X, float64(at.y*10)-enemy.Y) < 16 {
						reached = true
						break
					}
				}
				if !reached {
					t.Fatalf("mission %d room %d unreachable spawn %+v", level.ID, room, enemy)
				}
			}
		}
	}
	levels[0].Rooms[0].Obstacles[0].X = 0
	if Campaign()[0].Rooms[0].Obstacles[0].X == 0 {
		t.Fatal("campaign aliases caller's state")
	}
}

func TestCoverJumpAndEnemyBypass(t *testing.T) {
	r := NewRunAtLevel("cover", Build{HP: 200}, time.Now(), content.AbyssMobCatalog(), 2)
	r.Level.Rooms[0].Obstacles = []Obstacle{{400, 380, 80, 40}}
	a := Actor{X: 385, Y: 400}
	r.moveActor(&a, 20, 0, false)
	if a.X != 385 {
		t.Fatal("ground movement crossed cover")
	}
	a.Jump = .5
	r.moveActor(&a, 20, 0, false)
	if a.X != 405 {
		t.Fatal("jump did not cross low cover")
	}
	a.Jump = 0
	r.moveActor(&a, 0, 0, false)
	if contains(r.Arena().Obstacles[0], a.X, a.Y, 10) {
		t.Fatal("landed inside cover")
	}
	a = Actor{X: 520, Y: 400}
	for step := 0; step < 300; step++ {
		r.moveActor(&a, -2, clamp(400-a.Y, -1, 1), true)
	}
	if a.X > 350 {
		t.Fatalf("enemy stuck behind cover %+v", a)
	}
}

func TestEnemiesCanApproachAcrossEveryCampaignArena(t *testing.T) {
	for id := 1; id <= LevelCount; id++ {
		r := NewRunAtLevel("pursuit", Build{HP: 1e9}, time.Now(), content.AbyssMobCatalog(), id)
		for room := 0; room < 3; room++ {
			r.Room = room
			for _, spawn := range r.EncounterPlan[room] {
				// Exercise ground pursuit independently of the monster's ranged
				// attacks, spells, or intentional treasure-goblin fleeing.
				spawn.Kind = "goblin"
				spawn.Speed = 115
				r.Enemies = []Actor{spawn}
				for step := 0; step < 1200; step++ {
					r.enemyTick(0, 1.0/30)
					if math.Abs(r.Enemies[0].X-r.Player.X) < 70 && math.Abs(r.Enemies[0].Y-r.Player.Y) < 25 {
						break
					}
				}
				if math.Abs(r.Enemies[0].X-r.Player.X) > 70 || math.Abs(r.Enemies[0].Y-r.Player.Y) > 25 {
					t.Fatalf("mission %d room %d stuck enemy %+v", id, room, r.Enemies[0])
				}
			}
		}
	}
}

func TestHazardsWarnThenDamageOnceAndRespectJumpPause(t *testing.T) {
	r := NewRunAtLevel("hazard", Build{HP: 200}, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
	r.Level.Rooms[0].Hazards = []Hazard{{Obstacle: Obstacle{100, 380, 100, 60}, Kind: "ice", Period: 6, Duration: 1}}
	r.Clock = .8
	r.hazardTick()
	if r.Player.HP != 200 {
		t.Fatal("warning dealt damage")
	}
	r.Clock = 1.3
	r.hazardTick()
	hp := r.Player.HP
	if hp >= 200 || r.SkillTimers["slowed"] == 0 {
		t.Fatal("active frost failed")
	}
	r.hazardTick()
	if r.Player.HP != hp {
		t.Fatal("hazard damaged twice in same cooldown")
	}
	r.SkillTimers["hazard-0"] = 0
	r.Player.Jump = .5
	r.hazardTick()
	if r.Player.HP != hp {
		t.Fatal("jump did not evade hazard")
	}
	r.Paused = true
	clock := r.Clock
	r.Step(Input{}, time.Unix(101, 0))
	if r.Clock != clock {
		t.Fatal("pause advanced hazard clock")
	}
}

func TestHazardWarningEventEmittedOncePerCycle(t *testing.T) {
	r := NewRunAtLevel("hazard-warn-test", Build{HP: 200}, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
	r.Level.Rooms[0].Hazards = []Hazard{{Obstacle: Obstacle{100, 380, 100, 60}, Kind: "fire", Period: 5, Duration: 1}}

	// Clock at start of warning phase
	r.Clock = 0.1
	r.hazardTick()

	warnCount := 0
	for _, e := range r.Events {
		if e.Kind == "hazard_warning" {
			warnCount++
		}
	}
	if warnCount != 1 {
		t.Fatalf("expected exactly 1 hazard_warning event, got %d", warnCount)
	}

	// Repeated tick in same warning phase must not emit another warning event
	r.Clock = 0.5
	r.hazardTick()
	warnCount = 0
	for _, e := range r.Events {
		if e.Kind == "hazard_warning" {
			warnCount++
		}
	}
	if warnCount != 1 {
		t.Fatalf("warning event repeated within same warning phase: %d", warnCount)
	}
}

func TestSeamlessCampaignAdvanceKeepsReceiptAndCompletion(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	r := NewRunAtLevel("campaign", Build{HP: 200}, time.Now(), catalog, 10)
	r.Status = "cleared"
	r.FinishCheckpoint("advance", catalog)
	if r.Room != 1 || r.Level.ID != 10 {
		t.Fatal("did not advance room")
	}
	r.Room = 2
	r.Status = "cleared"
	r.BankedGold = 300
	r.BankedItems = []string{"Abyss blade"}
	r.Player.HP = 100
	r.FinishCheckpoint("advance", catalog)
	if r.Level.ID != 11 || r.Room != 0 || r.Status != "fighting" || r.ID != "campaign" || r.BankedGold != 300 || len(r.BankedItems) != 1 || r.Player.HP != 150 {
		t.Fatalf("lost campaign state %+v", r)
	}
	if len(r.CompletedLevels) != 1 || r.CompletedLevels[0] != 10 {
		t.Fatal("lost completed mission")
	}
	r.FinishCheckpoint("advance", catalog)
	if r.Level.ID != 11 {
		t.Fatal("advanced uncleared mission")
	}
	r.setLevel(100, catalog)
	r.Room = 2
	r.Status = "cleared"
	r.FinishCheckpoint("advance", catalog)
	if r.Status != "complete" || len(r.CompletedLevels) != 2 {
		t.Fatal("campaign did not end")
	}
	r.Status = "cleared"
	r.FinishCheckpoint("next", catalog)
	if len(r.CompletedLevels) != 2 {
		t.Fatal("duplicated completion")
	}
}
