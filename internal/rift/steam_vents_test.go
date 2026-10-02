package rift

import (
	"encoding/json"
	"testing"
)

func TestForgeSteamVentsAreSavedNonCombatScenery(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for _, arena := range level.Rooms {
			if level.Region != 1 && len(arena.SteamVents) != 0 {
				t.Fatal("steam authored outside forge")
			}
			if level.Region == 1 && len(arena.SteamVents) != 2 {
				t.Fatal("forge missing vents")
			}
			for _, v := range arena.SteamVents {
				count++
				if v.Y+v.H > 305 {
					t.Fatal("steam extends into combat floor")
				}
				for _, solid := range arena.solidObstacles() {
					if solid == v {
						t.Fatal("steam became solid")
					}
				}
			}
			raw, err := json.Marshal(arena)
			if err != nil {
				t.Fatal(err)
			}
			var saved Arena
			if err = json.Unmarshal(raw, &saved); err != nil {
				t.Fatal(err)
			}
			if len(saved.SteamVents) != len(arena.SteamVents) {
				t.Fatal("saved vents lost")
			}
		}
	}
	if count != 60 {
		t.Fatalf("got %d vents", count)
	}
}
