package rift

import (
	"testing"
	"time"
	"ts3news/internal/content"
)

func overlappingHazardRun() *Run {
	r := NewRunAtLevel("hazard-safety", Build{HP: 200}, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
	r.Level.Rooms[0].Hazards = nil
	for _, kind := range []string{"fire", "ice", "void"} {
		r.Level.Rooms[0].Hazards = append(r.Level.Rooms[0].Hazards, Hazard{Obstacle: Obstacle{100, 380, 180, 60}, Kind: kind, Jumpable: true, Period: 6, Duration: 2})
	}
	r.Clock = 1.3
	return r
}

func TestDefeatedPlayerIgnoresHazardEffects(t *testing.T) {
	r := overlappingHazardRun()
	r.Player.HP = 0
	before := r.Player
	events := len(r.Events)
	r.hazardTick()
	if r.Player != before || r.SkillTimers["slowed"] > 0 || len(r.Events) != events {
		t.Fatal("hazards affected defeated player")
	}
	r = overlappingHazardRun()
	r.Player.HP = 1
	before = r.Player
	r.hazardTick()
	if r.Player.HP != 0 || r.Player.X != before.X || r.SkillTimers["slowed"] > 0 {
		t.Fatal("later hazards changed a fatally hit player")
	}
	if r.Stats.HitsTaken != 1 {
		t.Fatal("fatal hazard counted repeated hits")
	}
}

func TestOverlappingHazardsShareDamageCooldown(t *testing.T) {
	r := overlappingHazardRun()
	r.hazardTick()
	if r.Stats.HitsTaken != 1 {
		t.Fatalf("overlap caused %d simultaneous hits", r.Stats.HitsTaken)
	}
	r.Enemies = []Actor{{ID: "distant", X: 1400, Y: 500, HP: 1000, MaxHP: 1000, Cooldown: 10}}
	r.tick(Input{}, .34)
	if r.Stats.HitsTaken != 1 {
		t.Fatal("overlap bypassed shared cooldown")
	}
	r.tick(Input{}, .02)
	if r.Stats.HitsTaken != 2 {
		t.Fatal("next hazard failed after recovery window")
	}
}
