package rift

import (
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestEveryCampaignRoomRequiresClearanceBeforeAdvancement(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	for mission := 1; mission <= LevelCount; mission++ {
		for room := range Rooms {
			for _, kind := range []string{"bank", "exit", "next", "advance"} {
				r := NewRunAtLevel("clearance", Build{HP: 300}, time.Unix(100, 0), catalog, mission)
				r.Room = room
				r.spawnRoom()
				r.Drops = []Drop{{ID: "pending", Gold: 30, Collected: true}}
				if r.NextRoom() {
					t.Fatalf("mission %d room %d advanced during fight", mission, room)
				}
				r.FinishCheckpoint(kind, catalog)
				if r.Level.ID != mission || r.Room != room || r.Status != "fighting" || r.BankedGold != 0 || r.Drops[0].Banked || len(r.CompletedLevels) != 0 || r.History[mission].Completions != 0 {
					t.Fatalf("mission %d room %d accepted %s before clearance", mission, room, kind)
				}
			}
		}
	}
}
