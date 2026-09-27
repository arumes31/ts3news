package bot

import (
	"testing"
	"time"
	"ts3news/internal/content"

	"ts3news/internal/rift"
)

func TestRiftDecodeRejectsInvalidActorPositions(t *testing.T) {
	selectors := map[string]func(*rift.Run) *rift.Actor{
		"player":     func(r *rift.Run) *rift.Actor { return &r.Player },
		"enemy":      func(r *rift.Run) *rift.Actor { return &r.Enemies[0] },
		"future":     func(r *rift.Run) *rift.Actor { return &r.EncounterPlan[0][0] },
		"boss reset": func(r *rift.Run) *rift.Actor { return r.Practice.BossStart },
		"lantern":    func(r *rift.Run) *rift.Actor { return r.RoomObjective.Lantern },
		"escort":     func(r *rift.Run) *rift.Actor { return r.RoomObjective.Escort },
		"ward":       func(r *rift.Run) *rift.Actor { return &r.RoomObjective.Lanes[0].Ward },
		"wave":       func(r *rift.Run) *rift.Actor { return &r.RoomObjective.Waves[0][0] },
	}
	for name, selectActor := range selectors {
		for axis, change := range map[string]func(*rift.Actor){
			"left": func(a *rift.Actor) { a.X = -1 }, "right": func(a *rift.Actor) { a.X = rift.Width + 1 },
			"above": func(a *rift.Actor) { a.Y = -1 }, "below": func(a *rift.Actor) { a.Y = 541 },
			"target": func(a *rift.Actor) { a.TargetX = 1e100 }, "route": func(a *rift.Actor) { a.RouteY = -1 },
		} {
			t.Run(name+"/"+axis, func(t *testing.T) {
				r := riftVitalsFixture()
				change(selectActor(r))
				saved, err := encodeRift(r)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = decodeRift(saved); err == nil {
					t.Fatal("invalid actor position accepted")
				}
			})
		}
	}
}

func TestRiftDecodePreservesPositionBoundaries(t *testing.T) {
	for _, point := range [][2]float64{{35, 315}, {rift.Width - 35, 490}, {160.25, 410.75}} {
		r := riftVitalsFixture()
		r.Player.X, r.Player.Y = point[0], point[1]
		r.Enemies[0].Kind = "generator"
		r.Enemies[0].X, r.Enemies[0].Y = 500, 290
		saved, err := encodeRift(r)
		if err != nil {
			t.Fatal(err)
		}
		got, err := decodeRift(saved)
		if err != nil {
			t.Fatal(err)
		}
		if got.Player.X != point[0] || got.Player.Y != point[1] || got.Enemies[0].Y != 290 {
			t.Fatal("valid position changed")
		}
	}
	for _, point := range [][2]float64{{34, 410}, {rift.Width - 34, 410}, {160, 314}, {160, 491}} {
		r := riftVitalsFixture()
		r.Player.X, r.Player.Y = point[0], point[1]
		saved, err := encodeRift(r)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = decodeRift(saved); err == nil {
			t.Fatal("player outside playable bounds accepted")
		}
	}
}

// Exercise saved navigation/target coordinates after simulation, not only spawns.
func TestRiftDecodePositionsAfterCampaignMovement(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	for level := 1; level <= rift.LevelCount; level++ {
		now := time.Unix(100, 0)
		r := rift.NewRunAtLevel("position-movement", rift.Build{HP: 100000, Damage: 10}, now, catalog, level)
		for room := 0; room < len(rift.Rooms); room++ {
			for tick := 0; tick < 60; tick++ {
				now = now.Add(100 * time.Millisecond)
				direction := 1.0
				if tick >= 30 {
					direction = -1
				}
				r.Step(rift.Input{X: direction, Y: direction, Attack: true, Jump: tick%12 == 0}, now)
				if tick%20 == 19 {
					saved, err := encodeRift(r)
					if err != nil {
						t.Fatal(err)
					}
					r, err = decodeRift(saved)
					if err != nil {
						t.Fatalf("level %d room %d tick %d: %v", level, room, tick, err)
					}
				}
			}
			r.Status = "cleared"
			if room < len(rift.Rooms)-1 && !r.NextRoom() {
				t.Fatal("failed to advance fixture")
			}
		}
	}
}
