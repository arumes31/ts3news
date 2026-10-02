package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func assertSpawnOutsideHazards(t *testing.T, arena Arena, a Actor) {
	t.Helper()
	for _, h := range arena.Hazards {
		if !h.Disabled && contains(h.Obstacle, a.X, a.Y, actorClearance(&a)) {
			t.Fatalf("%s spawned in %s footprint at %.0f/%.0f", a.ID, h.Kind, a.X, a.Y)
		}
	}
	for _, o := range arena.solidObstacles() {
		if contains(o, a.X, a.Y, actorClearance(&a)) {
			t.Fatalf("%s spawned in cover", a.ID)
		}
	}
}

func TestEveryCampaignEnemySpawnAvoidsHazardFootprints(t *testing.T) {
	for _, level := range Campaign() {
		r := NewRunAtLevel("spawn-hazards", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level.ID)
		for room, arena := range level.Rooms {
			for _, a := range r.EncounterPlan[room] {
				assertSpawnOutsideHazards(t, arena, a)
			}
			r.Room = room
			r.spawnRoom()
			for _, a := range r.Enemies {
				if !a.isObjectiveProp() {
					assertSpawnOutsideHazards(t, arena, a)
				}
			}
		}
	}
}

func TestSpawnPlacementHonorsBossFootprintAndPreservesSafePositions(t *testing.T) {
	arena := Arena{Hazards: []Hazard{{Obstacle: Obstacle{500, 360, 100, 60}, Kind: "fire", Period: 7, Duration: 1}}, HighCover: []Obstacle{{470, 320, 30, 140}}}
	for _, kind := range []string{"boss", "goblin", "wolf"} {
		a := Actor{ID: kind, Kind: kind, X: 590, Y: 410, HP: 200, MaxHP: 200}
		if !arena.settleEnemySpawn(&a) {
			t.Fatal("no safe spawn")
		}
		assertSpawnOutsideHazards(t, arena, a)
		before := a
		if !arena.settleEnemySpawn(&a) || a != before {
			t.Fatal("safe spawn changed")
		}
		data, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		var restored Actor
		if err = json.Unmarshal(data, &restored); err != nil {
			t.Fatal(err)
		}
		if !arena.settleEnemySpawn(&restored) || restored != before {
			t.Fatal("saved spawn changed")
		}
	}
	a := Actor{X: 550, Y: 390}
	arena.Hazards[0].Disabled = true
	if !arena.settleEnemySpawn(&a) || a.X != 550 || a.Y != 390 {
		t.Fatal("disabled hazard moved safe actor")
	}
}

func TestWaveReinforcementAvoidsHazardAtArrival(t *testing.T) {
	r := dropEdgeTestRun()
	r.Level.Rooms[0] = Arena{Hazards: []Hazard{{Obstacle: Obstacle{500, 360, 100, 60}, Kind: "fire", Period: 7, Duration: 1}}}
	r.Enemies = []Actor{{ID: "fallen", HP: 0, MaxHP: 100}}
	r.RoomObjective = &RoomObjective{Kind: "survive_waves", Target: 3, Wave: 1, NextWaveSeconds: .01, Waves: [][]Actor{nil, {{ID: "reinforcement", Kind: "boss", X: 550, Y: 390, HP: 200, MaxHP: 200}}, nil}}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	restored.tickWaveObjective(.02)
	if len(restored.Enemies) != 2 || restored.RoomObjective.Wave != 2 {
		t.Fatal("wave failed to arrive")
	}
	enemy := restored.Enemies[1]
	assertSpawnOutsideHazards(t, restored.Arena(), enemy)
	if enemy.HP != 200 || enemy.Cooldown < 1.2 {
		t.Fatal("spawn changed health or removed arrival grace")
	}
	if restored.RoomObjective.Waves[1][0].X != 550 {
		t.Fatal("arrival rewrote saved wave template")
	}
}
