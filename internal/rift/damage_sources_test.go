package rift

import (
	"encoding/json"
	"math"
	"testing"
)

func TestDamageSourcesCountOnlyActualHealthLoss(t *testing.T) {
	for _, mode := range []string{"normal", "guard", "barrier", "lethal"} {
		t.Run(mode, func(t *testing.T) {
			r := circleTestRun()
			r.Build.Armor = 10
			if mode == "guard" {
				r.Player.Guard = true
			}
			if mode == "barrier" {
				r.Barrier = 100
			}
			if mode == "lethal" {
				r.Player.HP = 3
			}
			before := r.Player.HP
			r.hurtPlayerFromHazard(30, r.Player.X, r.Player.Y)
			lost := before - r.Player.HP
			if math.Abs(r.Stats.HazardDamageTaken-lost) > 1e-8 || r.Stats.EnemyDamageTaken != 0 {
				t.Fatal("hazard attribution includes blocked or excess damage")
			}
			before = r.Player.HP
			r.hurtPlayerFromEnemy(20, r.Player.X, r.Player.Y, "enemy")
			enemyLoss := before - r.Player.HP
			if math.Abs(r.Stats.EnemyDamageTaken-enemyLoss) > 1e-8 || math.Abs(r.Stats.DamageTaken-lost-enemyLoss) > 1e-8 {
				t.Fatal("enemy attribution contaminated hazard damage")
			}
		})
	}
}
func TestAllFloorHazardsRecordDamageSource(t *testing.T) {
	for _, kind := range []string{"fire", "ice", "poison", "thorns", "rune", "radiant", "void"} {
		r := circleTestRun()
		r.Build.Armor = 0
		r.Clock = 1.3
		r.Player.X = 500
		r.Player.Y = 400
		r.Level.Rooms[0] = Arena{Hazards: []Hazard{{Obstacle: Obstacle{480, 380, 80, 40}, Kind: kind, Period: 7, Duration: 1}}}
		r.hazardTick()
		if r.Stats.HazardDamageTaken <= 0 || r.Stats.HazardDamageTaken != r.Stats.DamageTaken || r.Stats.EnemyDamageTaken != 0 {
			t.Fatalf("%s source missing", kind)
		}
	}
}
func TestDamageSourcesPersistAndRoomSummariesUseDeltas(t *testing.T) {
	r := circleTestRun()
	r.Build.Armor = 0
	r.hurtPlayerFromHazard(10, r.Player.X, r.Player.Y)
	r.hurtPlayerFromEnemy(20, r.Player.X, r.Player.Y, "enemy")
	r.spawnRoom()
	r.hurtPlayerFromHazard(5, r.Player.X, r.Player.Y)
	r.hurtPlayerFromEnemy(7, r.Player.X, r.Player.Y, "enemy")
	r.RecordEncounterSummary("cleared")
	if r.LastEncounter.HazardDamageTaken != 5 || r.LastEncounter.EnemyDamageTaken != 7 {
		t.Fatal("room summary includes prior rooms")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Stats.HazardDamageTaken != 15 || saved.Stats.EnemyDamageTaken != 27 || saved.LastEncounter.HazardDamageTaken != 5 {
		t.Fatal("source records lost on save")
	}
}

func TestCollapseDamageUsesHazardSource(t *testing.T) {
	r := collapseTestRun()
	r.Player.X = 50
	r.Build.Armor = 0
	r.tickCollapseObjective(4)
	if r.Stats.HazardDamageTaken <= 0 || r.Stats.HazardDamageTaken != r.Stats.DamageTaken || r.Stats.EnemyDamageTaken != 0 {
		t.Fatal("collapse damage not attributed to hazard")
	}
}
