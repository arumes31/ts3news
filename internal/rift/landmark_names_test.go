package rift

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCampaignRoomsHaveDistinctLandmarkNames(t *testing.T) {
	seen := map[string]bool{}
	for _, level := range Campaign() {
		for _, room := range level.Rooms {
			if seen[room.Name] {
				t.Fatalf("room landmark reused: %s", room.Name)
			}
			seen[room.Name] = true
			parts := strings.Split(room.Name, " / ")
			if len(parts) != 2 || parts[1] == "" || parts[1] == "Approach" || parts[1] == "Inner Court" || parts[1] == "Guardian's Stand" {
				t.Fatalf("room has no specific landmark: %s", room.Name)
			}
			data, err := json.Marshal(room)
			if err != nil {
				t.Fatal(err)
			}
			var saved Arena
			if err = json.Unmarshal(data, &saved); err != nil || saved.Name != room.Name {
				t.Fatal("room name changed on save")
			}
		}
	}
	if len(seen) != 300 {
		t.Fatalf("got %d landmark rooms", len(seen))
	}
	var legacy Arena
	if err := json.Unmarshal([]byte(`{"name":"Pilgrim's Gate / Approach"}`), &legacy); err != nil || legacy.Name != "Pilgrim's Gate / Approach" {
		t.Fatal("legacy name replaced")
	}
}
