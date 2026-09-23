package rift

import (
	"encoding/json"
	"testing"
)

func TestCampaignRoomCameraFramingAndSaveCompatibility(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for _, arena := range level.Rooms {
			want := 0.0
			switch arena.Objective {
			case "split_defense":
				want = 430
			case "escort_spirit":
				want = 280
			}
			if arena.CameraLead != want {
				t.Fatalf("mission %d %s framing %v, want %v", level.ID, arena.Name, arena.CameraLead, want)
			}
			if want > 0 {
				count++
			}
			data, err := json.Marshal(arena)
			if err != nil {
				t.Fatal(err)
			}
			var restored Arena
			if err = json.Unmarshal(data, &restored); err != nil || restored.CameraLead != want {
				t.Fatalf("framing changed on save: %v", err)
			}
		}
	}
	if count != 20 {
		t.Fatalf("authored %d framed rooms, want20", count)
	}
	var legacy Arena
	if err := json.Unmarshal([]byte(`{"name":"Legacy courtyard"}`), &legacy); err != nil || legacy.CameraLead != 0 {
		t.Fatal("legacy camera default changed")
	}
}
