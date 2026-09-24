package rift

import (
	"testing"
)

func TestEveryCampaignRoomHasPermanentHazardSafeGround(t *testing.T) {
	rooms := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			rooms++
			if !permanentHazardSafeRoute(arena) {
				t.Fatalf("mission %d tier %d lacks a permanent safe entrance-to-exit route", level.ID, room+1)
			}
		}
	}
	if rooms != 300 {
		t.Fatalf("checked %d rooms, want 300", rooms)
	}
}

func TestSafeGroundValidatorRejectsCoverageAndDisconnectedRefuges(t *testing.T) {
	for _, tc := range []struct {
		name    string
		hazards []Hazard
		want    bool
	}{
		{"open ground", nil, true},
		{"isolated puddle", []Hazard{{Obstacle: Obstacle{500, 350, 100, 40}}}, true},
		{"whole floor", []Hazard{{Obstacle: Obstacle{35, 315, 1530, 175}}}, false},
		{"safe pockets separated by danger", []Hazard{{Obstacle: Obstacle{700, 315, 10, 175}}}, false},
		{"disabled barrier", []Hazard{{Obstacle: Obstacle{700, 315, 10, 175}, Disabled: true}}, true},
		{"thin strip between grid samples", []Hazard{{Obstacle: Obstacle{705, 315, 1, 175}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := permanentHazardSafeRoute(Arena{Hazards: tc.hazards}); got != tc.want {
				t.Fatalf("route=%v want %v", got, tc.want)
			}
		})
	}
}
