package rift

import (
	"encoding/json"
	"math"
	"testing"
)

func waveGateTestRun() *Run {
	r := waveTestRun()
	r.Enemies = nil
	for _, group := range r.RoomObjective.Waves {
		r.Enemies = append(r.Enemies, group...)
	}
	r.RoomObjective = nil
	r.Level.Rooms[0].WaveGate = &Obstacle{840, 370, 24, 50}
	r.beginWaveObjective()
	return r
}

func TestWaveGateWarningSaveAndCycle(t *testing.T) {
	r := waveGateTestRun()
	if r.RoomObjective.Gate == nil || r.RoomObjective.Gate.Closed || r.RoomObjective.Gate.CloseIn != .8 {
		t.Fatal("missing warning")
	}
	r.tickWaveObjective(.4)
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	saved.Paused = true
	saved.tickWaveObjective(1)
	if saved.RoomObjective.Gate.CloseIn != .4 {
		t.Fatal("paused gate advanced")
	}
	saved.Paused = false
	saved.tickWaveObjective(.4)
	if !saved.RoomObjective.Gate.Closed {
		t.Fatal("gate did not close")
	}
	if len(saved.Arena().HighCover) != len(saved.Level.Rooms[0].HighCover)+1 {
		t.Fatal("closed gate missing from collision")
	}
	before := len(saved.Level.Rooms[0].HighCover)
	a := saved.Arena()
	a.HighCover[len(a.HighCover)-1].X = 0
	if len(saved.Level.Rooms[0].HighCover) != before || saved.RoomObjective.Gate.X != 840 {
		t.Fatal("gate aliases frozen arena")
	}
	for i := range saved.Enemies {
		saved.Enemies[i].HP = 0
	}
	saved.tickWaveObjective(.01)
	if saved.RoomObjective.Gate.Closed || saved.RoomObjective.Gate.CloseIn != 0 {
		t.Fatal("intermission gate did not open")
	}
	saved.tickWaveObjective(2.5)
	if saved.RoomObjective.Gate.Closed || saved.RoomObjective.Gate.CloseIn != .8 {
		t.Fatal("next wave warning missing")
	}
	saved.tickWaveObjective(.8)
	if !saved.RoomObjective.Gate.Closed {
		t.Fatal("next wave gate did not close")
	}
}

func TestWaveGateNeverClosesOnLivingActors(t *testing.T) {
	for _, kind := range []string{"player", "boss", "goblin"} {
		t.Run(kind, func(t *testing.T) {
			r := waveGateTestRun()
			target := &r.Player
			if kind != "player" {
				target = &r.Enemies[0]
				target.Kind = kind
			}
			target.X, target.Y, target.Jump = 840-actorClearance(target)+1, 395, 1
			before := *target
			r.tickWaveObjective(2)
			if r.RoomObjective.Gate.Closed || target.X != before.X || target.Y != before.Y || target.HP != before.HP {
				t.Fatal("gate crushed or moved occupant")
			}
			target.X = 160
			r.tickWaveObjective(.01)
			if !r.RoomObjective.Gate.Closed {
				t.Fatal("clear threshold never closed")
			}
		})
	}
}

func TestWaveGateBlocksJumpAndShotsButOpensBetweenWaves(t *testing.T) {
	r := waveGateTestRun()
	r.tickWaveObjective(.8)
	r.Player.X, r.Player.Y, r.Player.Jump = 800, 395, 1
	r.moveActor(&r.Player, 100, 0, false)
	if r.Player.X > 830 {
		t.Fatal("jump crossed gate")
	}
	if hit, _ := r.projectileCoverImpact(800, 395, 900, 395); hit > 1 {
		t.Fatal("shot crossed gate")
	}
	for i := range r.Enemies {
		r.Enemies[i].HP = 0
	}
	r.tickWaveObjective(.01)
	r.moveActor(&r.Player, 100, 0, false)
	if r.Player.X < 900 {
		t.Fatal("open gate blocked movement")
	}
}

func TestAuthoredWaveGatesKeepAllWalkableCellsConnected(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for _, arena := range level.Rooms {
			if arena.WaveGate == nil {
				continue
			}
			count++
			walls := append(append([]Obstacle{}, arena.solidObstacles()...), *arena.WaveGate)
			for _, radius := range []float64{6, 10, 18} {
				// Flood all legal 5px cells at every supported collision footprint. Both flanks
				// must remain connected; this checks more than just the initial spawn points.
				type cell struct{ x, y int }
				free := map[cell]bool{}
				for x := 35; x <= 1565; x += 5 {
					for y := 315; y <= 490; y += 5 {
						blocked := false
						for _, w := range walls {
							blocked = blocked || contains(w, float64(x), float64(y), radius)
						}
						if !blocked {
							free[cell{x, y}] = true
						}
					}
				}
				seed := cell{160, 335}
				if !free[seed] {
					t.Fatal("blocked entrance lane")
				}
				seen := map[cell]bool{seed: true}
				queue := []cell{seed}
				for head := 0; head < len(queue); head++ {
					p := queue[head]
					for _, d := range []cell{{5, 0}, {-5, 0}, {0, 5}, {0, -5}} {
						n := cell{p.x + d.x, p.y + d.y}
						if free[n] && !seen[n] {
							seen[n] = true
							queue = append(queue, n)
						}
					}
				}
				if len(seen) != len(free) {
					t.Fatalf("mission %d gate isolates %d cells", level.ID, len(free)-len(seen))
				}
			}
		}
	}
	if count != 10 {
		t.Fatalf("got %d gated rooms", count)
	}
	levels := Campaign()
	levels[4].Rooms[1].WaveGate.X = 0
	if Campaign()[4].Rooms[1].WaveGate.X == 0 {
		t.Fatal("gate definition aliases campaign")
	}
	r := waveTestRun()
	if r.RoomObjective.Gate != nil {
		t.Fatal("legacy arena gained a gate")
	}
}

func TestWaveGateSweptMovementAndBoundaryRelease(t *testing.T) {
	for _, jump := range []float64{0, .5} {
		for _, v := range [][4]float64{{800, 395, 100, 0}, {900, 395, -100, 0}, {852, 330, 0, 120}, {852, 460, 0, -120}} {
			r := waveGateTestRun()
			r.tickWaveObjective(.8)
			r.Player.X, r.Player.Y, r.Player.Jump = v[0], v[1], jump
			r.moveActor(&r.Player, v[2], v[3], false)
			if r.Player.X != v[0] || r.Player.Y != v[1] {
				t.Fatalf("crossed gate from %v, jump %v", v, jump)
			}
		}
	}
	r := waveGateTestRun()
	r.tickWaveObjective(.8)
	r.Player.X, r.Player.Y = 830, 395
	r.moveActor(&r.Player, -10, 0, false)
	if r.Player.X != 820 {
		t.Fatal("actor stuck to gate edge")
	}
	for _, mode := range []string{"dead", "ended", "paused"} {
		r := waveGateTestRun()
		g := *r.RoomObjective.Gate
		switch mode {
		case "dead":
			r.Player.HP = 0
		case "ended":
			r.Status = "defeated"
		case "paused":
			r.Paused = true
		}
		r.tickWaveObjective(10)
		if *r.RoomObjective.Gate != g {
			t.Fatalf("inactive %s gate changed", mode)
		}
	}
}

func TestWaveGatePursuersNavigateBothDirections(t *testing.T) {
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			if arena.WaveGate == nil {
				continue
			}
			for _, kind := range []string{"wolf", "goblin", "boss"} {
				for _, direction := range []float64{-1, 1} {
					r := waveGateTestRun()
					r.Level = &level
					r.Room = room
					r.RoomObjective.Gate = &WaveGate{Obstacle: *arena.WaveGate, Closed: true}
					g := arena.WaveGate
					from, to := g.X-50, g.X+g.W+50
					if direction < 0 {
						from, to = to, from
					}
					a := Actor{ID: "pursuer", Kind: kind, HP: 100, X: from, Y: g.Y + g.H/2}
					targetY := a.Y
					for step := 0; step < 200; step++ {
						r.moveActor(&a, clamp(to-a.X, -4, 4), clamp(targetY-a.Y, -4, 4), true)
					}
					if math.Abs(a.X-to) > 5 || math.Abs(a.Y-targetY) > 5 {
						t.Fatalf("mission %d %s direction %v stuck at %.1f,%.1f", level.ID, kind, direction, a.X, a.Y)
					}
				}
			}
		}
	}
}
