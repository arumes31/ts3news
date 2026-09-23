package rift

import (
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestEveryCampaignLowCoverHasVerticalJumpClearance(t *testing.T) {
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			for index, o := range arena.Obstacles {
				for _, direction := range []float64{-1, 1} {
					r := NewRunAtLevel("vault-clearance", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level.ID)
					r.Room = room
					r.RoomObjective = nil
					r.Level.Rooms[room].Hazards = nil
					r.Enemies = []Actor{{ID: "watcher", Kind: "goblin", X: 1500, Y: 490, HP: 100, MaxHP: 100, Knockdown: 100}}
					r.Player.X = o.X + o.W/2
					margin := actorClearance(&r.Player) + 1
					if direction > 0 {
						r.Player.Y = o.Y - margin
					} else {
						r.Player.Y = o.Y + o.H + margin
					}
					start := r.Player.Y
					r.tick(Input{Y: direction}, .02)
					if r.Player.Y != start {
						t.Fatalf("mission %d room %d cover %d walking entered low cover", level.ID, room, index)
					}
					r.tick(Input{Jump: true}, .02)
					for n := 0; n < 34; n++ {
						r.tick(Input{Y: direction}, .02)
					}
					passed := direction > 0 && r.Player.Y >= o.Y+o.H+actorClearance(&r.Player) || direction < 0 && r.Player.Y <= o.Y-actorClearance(&r.Player)
					if !passed {
						t.Fatalf("mission %d room %d cover %d direction %.0f vault stopped at %.2f", level.ID, room, index, direction, r.Player.Y)
					}
				}
			}
		}
	}
}
