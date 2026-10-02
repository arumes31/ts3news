package rift

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"ts3news/internal/content"
)

func TestOffsetBastionFlanksAndSavedGeometry(t *testing.T) {
	for id := 3; id <= LevelCount; id += 10 {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			r := NewRunAtLevel("offset-bastion", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), id)
			r.Room = 2
			arena := r.Arena()
			if !strings.Contains(arena.Name, "Offset Bastion") {
				t.Fatal("final tier has no offset bastion")
			}
			walls := arena.HighCover
			if len(walls) != 3 || walls[0].H < 80 || walls[0].W <= walls[2].W {
				t.Fatal("missing asymmetric central defenses")
			}
			start := Actor{ID: "player", X: walls[0].X - 60, Y: 410}
			endX := walls[2].X + walls[2].W + 60
			direct := start
			for i := 0; i < 400; i++ {
				r.moveActor(&direct, 3, 0, false)
			}
			if direct.X >= walls[0].X {
				t.Fatal("direct approach crosses central defense")
			}
			// Use the closest straight corridor on each side of the complete
			// defense footprint, with two spare units beyond the player's body.
			upper, lower := math.Inf(1), math.Inf(-1)
			for _, wall := range walls {
				upper = math.Min(upper, wall.Y-actorClearance(&start)-2)
				lower = math.Max(lower, wall.Y+wall.H+actorClearance(&start)+2)
			}
			lengths := []float64{}
			for _, lane := range []float64{upper, lower} {
				actor := start
				distance := 0.0
				for _, point := range [][2]float64{{start.X, lane}, {endX, lane}, {endX, start.Y}} {
					for step := 0; step < 600 && math.Hypot(actor.X-point[0], actor.Y-point[1]) > .001; step++ {
						x, y := actor.X, actor.Y
						r.moveActor(&actor, clamp(point[0]-x, -2, 2), clamp(point[1]-y, -2, 2), false)
						distance += math.Hypot(actor.X-x, actor.Y-y)
					}
					if math.Hypot(actor.X-point[0], actor.Y-point[1]) > .001 {
						t.Fatalf("flank %.0f blocked at %.0f/%.0f", lane, actor.X, actor.Y)
					}
				}
				lengths = append(lengths, distance)
			}
			if math.Abs(lengths[0]-lengths[1]) < 35 {
				t.Fatalf("approaches are not distinct: %v", lengths)
			}
			if failures := AuditArenaReachability(arena, Actor{ID: "entry", X: arena.Entrance.X, Y: arena.Entrance.Y}, r.EncounterPlan[2]); len(failures) > 0 {
				t.Fatalf("unreachable defenders: %+v", failures)
			}
			originalWall := walls[0]
			raw, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var saved Run
			if err = json.Unmarshal(raw, &saved); err != nil {
				t.Fatal(err)
			}
			r.Level.Rooms[2].HighCover[0].X += 123
			if saved.Arena().HighCover[0] != originalWall {
				t.Fatal("saved geometry did not freeze independently")
			}
		})
	}
}
