package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestArrivalClearsProjectilesAndDelaysEveryCampaignRangedEnemy(t *testing.T) {
	for level := 1; level <= 100; level++ {
		r := NewRunAtLevel("arrival", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level)
		for room := 0; room < len(Rooms); room++ {
			if room > 0 {
				r.Projectiles = []Projectile{{Enemy: true, X: 160, Y: 410, Life: 4}, {X: 160, Y: 410, Life: 4}}
				r.Status = "cleared"
				if !r.NextRoom() {
					t.Fatal("transition failed")
				}
			}
			if len(r.Projectiles) != 0 {
				t.Fatal("arrival retained projectiles")
			}
			for _, e := range r.Enemies {
				if (e.Kind == "archer" || e.Kind == "boss") && e.Cooldown < 1.5+rangedCooldownOffset(&e) {
					t.Fatalf("mission %d room %d: %s starts ready to fire: %f", level, room, e.Kind, e.Cooldown)
				}
			}
		}
	}
}

func TestArrivalDelayPersistsAndEventuallyAllowsArcherFire(t *testing.T) {
	r := circleTestRun()
	r.Level.Rooms[0] = Arena{}
	r.EncounterPlan[0] = []Actor{{ID: "arrival-archer", Kind: "archer", X: 420, Y: 410, HP: 100, MaxHP: 100, Damage: 10}}
	r.spawnRoom()
	r.RoomObjective = nil
	for n := 0; n < 30; n++ {
		r.tick(Input{}, .02)
	}
	saved, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err = json.Unmarshal(saved, &restored); err != nil {
		t.Fatal(err)
	}
	before := restored.Enemies[0].Cooldown
	restored.Paused = true
	restored.Step(Input{}, time.UnixMilli(restored.LastMS+200))
	if restored.Enemies[0].Cooldown != before {
		t.Fatal("pause consumed arrival delay")
	}
	restored.Paused = false
	for n := 0; n < 40; n++ {
		restored.tick(Input{}, .02)
		if len(restored.Projectiles) != 0 || restored.Enemies[0].Windup > 0 {
			t.Fatal("archer attacked during arrival delay")
		}
	}
	for n := 0; n < 150; n++ {
		restored.tick(Input{}, .02)
		if len(restored.Projectiles) > 0 {
			return
		}
	}
	t.Fatal("archer never resumed firing")
}
