package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func assertBossSpawnSpace(t *testing.T, arena Arena, actors []Actor) {
	t.Helper()
	for _, boss := range actors {
		if boss.Kind != "boss" || boss.HP <= 0 {
			continue
		}
		if boss.X < 49 || boss.X > Width-49 || boss.Y < 329 || boss.Y > 476 {
			t.Fatalf("boss against boundary: %+v", boss)
		}
		for _, o := range arena.solidObstacles() {
			if contains(o, boss.X, boss.Y, 32) {
				t.Fatal("boss lacks wall margin")
			}
		}
		for _, h := range arena.Hazards {
			if !h.Disabled && contains(h.Obstacle, boss.X, boss.Y, 32) {
				t.Fatal("boss lacks hazard margin")
			}
		}
		space := Obstacle{boss.X - 60, boss.Y - 32, 120, 64}
		for _, a := range actors {
			if a.ID != boss.ID && a.HP > 0 && !a.isObjectiveProp() && contains(space, a.X, a.Y, actorClearance(&a)) {
				t.Fatalf("%s crowds boss %s", a.ID, boss.ID)
			}
		}
	}
}
func TestCampaignReservesBossSpawnSpace(t *testing.T) {
	for _, level := range Campaign() {
		r := NewRunAtLevel("boss-space", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level.ID)
		for room, actors := range r.EncounterPlan {
			assertBossSpawnSpace(t, level.Rooms[room], actors)
			r.Room = room
			r.spawnRoom()
			assertBossSpawnSpace(t, r.Arena(), r.Enemies)
		}
	}
}
func TestCrowdedBossSpawnMovesFollowersAndPersists(t *testing.T) {
	r := circleTestRun()
	r.Level.Rooms[0] = Arena{HighCover: []Obstacle{{500, 360, 80, 70}}}
	r.EncounterPlan[0] = []Actor{{ID: "follower", Kind: "goblin", X: 600, Y: 400, HP: 100}, {ID: "boss", Kind: "boss", X: 600, Y: 400, HP: 200}, {ID: "safe", Kind: "goblin", X: 1000, Y: 410, HP: 100}}
	r.spawnRoom()
	assertBossSpawnSpace(t, r.Arena(), r.Enemies)
	if r.Enemies[2].X != 1000 || r.Enemies[2].Y != 410 {
		t.Fatal("unrelated safe spawn moved")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	assertBossSpawnSpace(t, saved.Arena(), saved.Enemies)
	for i, a := range r.Enemies {
		if saved.Enemies[i].X != a.X || saved.Enemies[i].Y != a.Y {
			t.Fatal("save changed reserved positions")
		}
	}
}

func TestBossWaveReservesSpaceBeforeFollowersWithoutChangingTemplate(t *testing.T) {
	r := circleTestRun()
	r.Level.Rooms[0] = Arena{}
	r.Enemies = []Actor{{ID: "fallen", HP: 0}}
	incoming := []Actor{{ID: "follower", Kind: "goblin", X: 600, Y: 400, HP: 100}, {ID: "boss", Kind: "boss", X: 600, Y: 400, HP: 200}}
	r.RoomObjective = &RoomObjective{Kind: "survive_waves", Target: 3, Wave: 1, NextWaveSeconds: .01, Waves: [][]Actor{nil, incoming, nil}}
	r.tickWaveObjective(.02)
	if len(r.Enemies) != 3 || r.RoomObjective.Wave != 2 {
		t.Fatal("wave did not arrive")
	}
	assertBossSpawnSpace(t, r.Arena(), r.Enemies)
	if incoming[0].X != 600 || incoming[0].Y != 400 {
		t.Fatal("placement mutated saved wave template")
	}
	if r.Enemies[1].Cooldown < 1.2 || r.Enemies[2].Cooldown < 1.2 {
		t.Fatal("arrival grace lost")
	}
}
