package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
	"ts3news/internal/content"
)

func dropEdgeTestRun() *Run {
	r := circleTestRun()
	r.RoomObjective = nil
	r.Level.Rooms[0] = Arena{DropEdges: []DropEdge{{ID: "ledge", X: 500, Y: 365, W: 180, LandingY: 425}}}
	r.Player.X, r.Player.Y = 580, 360
	return r
}

func TestDropEdgeAllowsDescentAndBlocksReturnEvenWhileJumping(t *testing.T) {
	for _, jump := range []float64{0, .5} {
		r := dropEdgeTestRun()
		r.Player.Jump = jump
		hp := r.Player.HP
		r.moveActor(&r.Player, 0, 10, false)
		if r.Player.Y != 425 || r.Player.HP != hp || r.Player.Jump != 0 {
			t.Fatal("drop did not land safely")
		}
		r.Player.Jump = jump
		r.moveActor(&r.Player, 0, -65, false)
		if r.Player.Y != 425 {
			t.Fatal("reverse movement crossed one-way edge")
		}
		r.Player.X = 480
		r.moveActor(&r.Player, 0, -65, false)
		if r.Player.Y != 360 {
			t.Fatal("route around edge blocked")
		}
	}
}

func TestDropEdgeLipCanBeLeftTowardUpperFloor(t *testing.T) {
	r := dropEdgeTestRun()
	r.Player.Y = 365
	r.moveActor(&r.Player, 0, -5, false)
	if r.Player.Y != 360 {
		t.Fatal("standing on upper lip prevented retreat")
	}
}

func TestDropEdgeRejectsUnsafeLanding(t *testing.T) {
	for _, mode := range []string{"low", "stone", "hazard", "bounds"} {
		t.Run(mode, func(t *testing.T) {
			r := dropEdgeTestRun()
			arena := &r.Level.Rooms[0]
			box := Obstacle{550, 410, 60, 35}
			switch mode {
			case "low":
				arena.Obstacles = []Obstacle{box}
			case "stone":
				arena.Cover = []TerrainCover{{Obstacle: box, ID: "stone", Material: "stone"}}
			case "hazard":
				arena.Hazards = []Hazard{{Obstacle: box, Kind: "fire", Period: 7, Duration: 1}}
			case "bounds":
				arena.DropEdges[0].LandingY = 510
			}
			r.moveActor(&r.Player, 0, 10, false)
			if r.Player.Y != 360 {
				t.Fatal("unsafe landing accepted")
			}
		})
	}
}

func TestDropEdgeNavigationReturnsByEitherEnd(t *testing.T) {
	for _, x := range []float64{520, 650} {
		r := dropEdgeTestRun()
		r.Player.X, r.Player.Y = x, 330
		e := Actor{ID: "pursuer", Kind: "goblin", X: x, Y: 425, HP: 100, MaxHP: 100, Speed: 90}
		for n := 0; n < 2000 && e.Y > 340; n++ {
			dx, dy := r.Player.X-e.X, r.Player.Y-e.Y
			r.moveActor(&e, math.Copysign(math.Min(math.Abs(dx), 1.8), dx), math.Copysign(math.Min(math.Abs(dy), 1.1), dy), true)
		}
		if e.Y > 340 {
			t.Fatalf("enemy trapped below edge at %.1f/%.1f", e.X, e.Y)
		}
	}
}

func TestDropEdgePersistsAndDoesNotAwardJumpOrLoot(t *testing.T) {
	r := dropEdgeTestRun()
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	saved.moveActor(&saved.Player, 0, 10, false)
	if saved.Player.Y != 425 || saved.Stats.Jumps != 0 || saved.Stats.Kills != 0 || len(saved.Drops) != 0 {
		t.Fatal("drop state or rewards changed")
	}
	drops := 0
	for _, event := range saved.Events {
		if event.Kind == "ledge_drop" {
			drops++
		}
	}
	if drops != 1 {
		t.Fatalf("drop events=%d", drops)
	}
}

func TestDropFaceBlocksSideEntryAndClimbing(t *testing.T) {
	r := dropEdgeTestRun()
	r.Player.X, r.Player.Y = 490, 400
	r.moveActor(&r.Player, 20, 0, false)
	if r.Player.X != 490 {
		t.Fatal("entered cliff face from side")
	}
	r.Player.X, r.Player.Y = 580, 430
	r.moveActor(&r.Player, 0, -10, false)
	if r.Player.Y != 430 {
		t.Fatal("climbed into cliff face")
	}
}

func TestDropNavigationPersistsUntilAboveLip(t *testing.T) {
	r := dropEdgeTestRun()
	a := Actor{ID: "pursuer", Kind: "goblin", X: 580, Y: 425, HP: 100, MaxHP: 100}
	r.moveActor(&a, 0, -2, true)
	if a.LedgeRoute != "ledge" {
		t.Fatal("detour not saved")
	}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var restored Actor
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 1000 && restored.Y > 340; n++ {
		r.moveActor(&restored, math.Copysign(math.Min(2, math.Abs(580-restored.X)), 580-restored.X), -2, true)
		if restored.Y > 365 && restored.Y < 425 && restored.X >= 500 && restored.X <= 680 {
			t.Fatal("detour entered face")
		}
	}
	if restored.Y > 340 {
		t.Fatal("saved detour trapped enemy")
	}
}

func TestDropFaceBlocksFastSideCrossing(t *testing.T) {
	r := dropEdgeTestRun()
	r.Player.X, r.Player.Y = 490, 400
	r.moveActor(&r.Player, 200, 0, false)
	if r.Player.X != 490 {
		t.Fatal("crossed entire cliff face in one movement")
	}
}

func TestDropNavigationCrossesAroundFaceFromEitherSide(t *testing.T) {
	for _, start := range []float64{480, 700} {
		r := dropEdgeTestRun()
		targetX := 1180 - start
		a := Actor{ID: "pursuer", Kind: "goblin", X: start, Y: 400, HP: 100, MaxHP: 100}
		for n := 0; n < 1500 && math.Hypot(a.X-targetX, a.Y-400) > 3; n++ {
			dx, dy := targetX-a.X, 400-a.Y
			r.moveActor(&a, clamp(dx, -2, 2), clamp(dy, -2, 2), true)
			if n%20 == 0 {
				data, err := json.Marshal(a)
				if err != nil {
					t.Fatal(err)
				}
				var restored Actor
				if err := json.Unmarshal(data, &restored); err != nil {
					t.Fatal(err)
				}
				a = restored
			}
			if a.X >= 500 && a.X <= 680 && a.Y > 365 && a.Y < 425 {
				t.Fatal("side detour entered cliff")
			}
		}
		if math.Hypot(a.X-targetX, a.Y-400) > 3 {
			t.Fatalf("side pursuer stuck at %.1f/%.1f", a.X, a.Y)
		}
	}
}

func TestCampaignDropLandingsAndBypassesAreClear(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			want := room == 0 && level.ID%10 == 1
			if (len(arena.DropEdges) == 1) != want {
				t.Fatalf("missing or unexpected drop in %d/%d", level.ID, room)
			}
			r := dropEdgeTestRun()
			r.Level = &level
			r.Room = room
			for _, edge := range arena.DropEdges {
				count++
				for x := edge.X - 30; x <= edge.X+edge.W+30; x += 2 {
					if !r.safeDropLanding(&r.Player, x, edge.LandingY) {
						t.Fatalf("unsafe landing %d x%.0f", level.ID, x)
					}
				}
				for _, x := range []float64{edge.X - 30, edge.X + edge.W + 30} {
					for y := edge.Y - 25; y <= edge.LandingY; y += 2 {
						if !r.safeDropLanding(&r.Player, x, y) {
							t.Fatalf("unsafe bypass %d %.0f/%.0f", level.ID, x, y)
						}
					}
				}
			}
		}
	}
	if count != 10 {
		t.Fatalf("drop count %d", count)
	}
}

func TestEverySubclassCanDropAndReturnUsingMovement(t *testing.T) {
	for _, style := range []string{"vanguard", "berserker", "marksman", "beastmaster", "elementalist", "chronomancer", "oracle", "geomancer", "bloodblade", "voidwalker", "runesmith", "alchemist"} {
		t.Run(style, func(t *testing.T) {
			r := dropEdgeTestRun()
			r.Build.Class = style
			r.Player.Kind = style
			r.Enemies = []Actor{{ID: "watcher", Kind: "goblin", X: 1450, Y: 485, HP: 100, MaxHP: 100, Knockdown: 1000}}
			hp := r.Player.HP
			for n := 0; n < 20 && r.Player.Y < 425; n++ {
				r.tick(Input{Y: 1}, .02)
			}
			if r.Player.Y < 425 || r.Player.HP != hp {
				t.Fatal("failed safe descent")
			}
			for n := 0; n < 20; n++ {
				r.tick(Input{Y: -1}, .02)
			}
			if r.Player.Y < 425 {
				t.Fatal("climbed one-way ledge")
			}
			for n := 0; n < 60 && r.Player.X > 480; n++ {
				r.tick(Input{X: -1}, .02)
			}
			for n := 0; n < 60 && r.Player.Y > 345; n++ {
				r.tick(Input{Y: -1}, .02)
			}
			if r.Player.X > 480 || r.Player.Y > 345 {
				t.Fatal("return path blocked")
			}
			if r.Stats.Jumps != 0 || r.Stats.Kills != 0 || len(r.Drops) != 0 {
				t.Fatal("terrain granted combat rewards")
			}
		})
	}
}

func TestCampaignDropRoutesReachUpperFloorAndAllSpawns(t *testing.T) {
	type point struct{ x, y int }
	for _, level := range Campaign() {
		if len(level.Rooms[0].DropEdges) == 0 {
			continue
		}
		r := NewRunAtLevel("drop-routes", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level.ID)
		edge := r.Arena().DropEdges[0]
		start := point{int(edge.X + edge.W/2), int(edge.LandingY)}
		queue := []point{start}
		seen := map[point]bool{start: true}
		for head := 0; head < len(queue); head++ {
			at := queue[head]
			for _, step := range []point{{5, 0}, {-5, 0}, {0, 5}, {0, -5}} {
				a := Actor{ID: "route-probe", X: float64(at.x), Y: float64(at.y)}
				r.moveActor(&a, float64(step.x), float64(step.y), false)
				next := point{int(a.X), int(a.Y)}
				if !seen[next] {
					seen[next] = true
					queue = append(queue, next)
				}
			}
		}
		if !seen[point{start.x, int(edge.Y) - 20}] || !seen[point{160, 410}] {
			t.Fatalf("mission %d has no return to upper floor or entrance", level.ID)
		}
		for _, e := range r.Enemies {
			reachable := false
			for at := range seen {
				if math.Hypot(float64(at.x)-e.X, float64(at.y)-e.Y) < 10 {
					reachable = true
					break
				}
			}
			if !reachable {
				t.Fatalf("mission %d unreachable %s", level.ID, e.ID)
			}
		}
		for at := range seen {
			if float64(at.x) >= edge.X && float64(at.x) <= edge.X+edge.W && float64(at.y) > edge.Y && float64(at.y) < edge.LandingY {
				t.Fatal("route graph enters cliff face")
			}
		}
	}
}
