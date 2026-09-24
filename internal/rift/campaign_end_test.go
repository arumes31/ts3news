package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestFullCampaignCheckpointsEndAtMission100AfterSaveAndReplay(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	r := NewRunAtLevel("full-campaign", Build{HP: 300}, time.Unix(100, 0), catalog, 1)
	roundTrip := func() {
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved Run
		if err = json.Unmarshal(data, &saved); err != nil {
			t.Fatal(err)
		}
		r = &saved
	}
	for mission := 1; mission <= LevelCount; mission++ {
		for room := range Rooms {
			if r.Level.ID != mission || r.Room != room || r.Status != "fighting" {
				t.Fatalf("unexpected mission %d room %d: %d/%d %s", mission, room, r.Level.ID, r.Room, r.Status)
			}
			r.Status = "cleared"
			for i := range r.Enemies {
				r.Enemies[i].HP = 0
			}
			r.Stats.Seconds += 10
			r.FinishCheckpoint("advance", catalog)
		}
		roundTrip()
		if len(r.CompletedLevels) != mission {
			t.Fatalf("mission %d completion count=%d", mission, len(r.CompletedLevels))
		}
	}
	if r.Level.ID != 100 || r.Room != 2 || r.Status != "complete" || r.Player.Pose != "victory" {
		t.Fatal("campaign did not stop at final boss victory")
	}
	for _, kind := range []string{"advance", "next", "bank", "exit", "advance"} {
		r.FinishCheckpoint(kind, catalog)
		roundTrip()
		if r.Level.ID != 100 || r.Room != 2 || r.Status != "complete" || len(r.CompletedLevels) != 100 {
			t.Fatal("final checkpoint replay advanced or duplicated completion")
		}
	}
	if len(r.History) != 100 {
		t.Fatalf("campaign history has %d missions", len(r.History))
	}
	for mission := 1; mission <= 100; mission++ {
		if r.CompletedLevels[mission-1] != mission || r.History[mission].Completions != 1 {
			t.Fatalf("mission %d not completed exactly once", mission)
		}
	}
	if _, exists := r.History[101]; exists {
		t.Fatal("mission 101 created")
	}
}
