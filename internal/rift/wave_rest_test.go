package rift

import (
	"encoding/json"
	"testing"
)

func waveRestTestRun() *Run {
	r := waveTestRun()
	r.Level.Rooms[0].RestAlcove = &Obstacle{70, 335, 210, 140}
	for i := range r.Enemies {
		r.Enemies[i].HP = 0
	}
	r.Player.X, r.Player.Y = 160, 410
	r.tickWaveObjective(.01)
	return r
}

func TestWaveRestHoldsCountdownAndSurvivesSave(t *testing.T) {
	r := waveRestTestRun()
	r.tickWaveObjective(20)
	if r.RoomObjective.NextWaveSeconds != 2.5 || r.RoomObjective.Wave != 1 {
		t.Fatal("resting did not hold wave")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	saved.tickWaveObjective(20)
	if !saved.inWaveRestAlcove() || saved.RoomObjective.NextWaveSeconds != 2.5 {
		t.Fatal("saved rest lost")
	}
	saved.Player.X = 320
	saved.Player.Guard = true
	saved.tickWaveObjective(20)
	if saved.inWaveRestAlcove() || saved.RoomObjective.NextWaveSeconds != 2.5 {
		t.Fatal("guard delay incorrectly grants safety or advances")
	}
	saved.Player.Guard = false
	saved.tickWaveObjective(1)
	if saved.RoomObjective.NextWaveSeconds != 1.5 {
		t.Fatal("leaving did not resume countdown")
	}
	saved.tickWaveObjective(1.5)
	if saved.RoomObjective.Wave != 2 || saved.inWaveRestAlcove() {
		t.Fatal("next wave missing")
	}
}

func TestWaveRestProtectsWithoutSpendingDefensesOrAwardingBlocks(t *testing.T) {
	r := waveRestTestRun()
	r.FirstHitGrace = true
	r.Barrier = 25
	r.Player.Guard = true
	hp, stats := r.Player.HP, r.Stats
	r.hurtPlayerFromEnemy(500, r.Player.X, r.Player.Y, "stray")
	r.hurtPlayerFromHazard(500, r.Player.X, r.Player.Y)
	if r.Player.HP != hp || r.Barrier != 25 || !r.FirstHitGrace || r.Stats.DamageTaken != stats.DamageTaken || r.Stats.Guards != stats.Guards {
		t.Fatal("rest spent health/defenses or awarded a guard")
	}
	r.Level.Rooms[0].Hazards = []Hazard{{Obstacle: Obstacle{100, 380, 120, 60}, Kind: "poison", Period: 5, Duration: 1}}
	r.Clock = 1.3
	r.hazardTick()
	if r.Player.HP != hp || r.SkillTimers["slowed"] > 0 {
		t.Fatal("hazard damaged or slowed resting player")
	}
	r.Player.X = 320
	r.Barrier = 0
	r.FirstHitGrace = false
	r.Player.Guard = false
	r.hurtPlayerFromEnemy(20, r.Player.X, r.Player.Y, "stray")
	if r.Player.HP >= hp {
		t.Fatal("safety leaked outside alcove")
	}
}

func TestWaveRestNeverProtectsAnActiveWaveOrLegacyArena(t *testing.T) {
	for _, mode := range []string{"alive", "no_interval", "complete", "ended", "no_alcove", "final_wave", "dead"} {
		t.Run(mode, func(t *testing.T) {
			r := waveRestTestRun()
			switch mode {
			case "alive":
				r.Enemies[0].HP = 1
			case "no_interval":
				r.RoomObjective.NextWaveSeconds = 0
			case "complete":
				r.RoomObjective.Complete = true
			case "ended":
				r.Status = "banked"
			case "final_wave":
				r.RoomObjective.Wave = r.RoomObjective.Target
			case "dead":
				r.Player.HP = 0
			case "no_alcove":
				r.Level.Rooms[0].RestAlcove = nil
			}
			if r.inWaveRestAlcove() {
				t.Fatal("invalid refuge immunity")
			}
		})
	}
}

func TestAuthoredWaveRestAlcovesAreClearAndDetached(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for _, arena := range level.Rooms {
			if arena.RestAlcove == nil {
				continue
			}
			count++
			if arena.Objective != "survive_waves" {
				t.Fatal("rest outside waves")
			}
			rest := *arena.RestAlcove
			for _, wall := range arena.solidObstacles() {
				if rest.X < wall.X+wall.W && rest.X+rest.W > wall.X && rest.Y < wall.Y+wall.H && rest.Y+rest.H > wall.Y {
					t.Fatal("rest overlaps collision")
				}
			}
			for _, h := range arena.Hazards {
				if rest.X < h.X+h.W && rest.X+rest.W > h.X && rest.Y < h.Y+h.H && rest.Y+rest.H > h.Y {
					t.Fatal("rest overlaps authored hazard")
				}
			}
			if !contains(rest, arena.Entrance.X, arena.Entrance.Y, 0) {
				t.Fatal("entrance outside refuge")
			}
		}
	}
	if count != 10 {
		t.Fatalf("got %d rest alcoves", count)
	}
	levels := Campaign()
	levels[4].Rooms[1].RestAlcove.X = 0
	if Campaign()[4].Rooms[1].RestAlcove.X == 0 {
		t.Fatal("rest geometry aliases")
	}
}
