package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
	"ts3news/internal/content"
)

func gateTestRun() *Run {
	r := circleTestRun()
	r.Level.Rooms[0].Objective = "rune_gate"
	r.beginRoomObjective()
	return r
}
func touchGateRune(r *Run, id int) {
	r.Player.X, r.Player.Y = 160, 410
	r.tickRuneGateObjective()
	for _, p := range r.RoomObjective.Pickups {
		if p.ID == id {
			r.Player.X, r.Player.Y = p.X, p.Y
		}
	}
	r.tickRuneGateObjective()
}
func TestRuneGateRequiresPatrolThenCorrectSequence(t *testing.T) {
	r := gateTestRun()
	if r.RoomObjective == nil {
		t.Fatal("gate missing")
	}
	o := r.RoomObjective
	touchGateRune(r, o.Sequence[0])
	if o.Collected != 0 {
		t.Fatal("gate accepted rune during combat")
	}
	r.Enemies[0].HP = 0
	for _, id := range o.Sequence {
		touchGateRune(r, id)
	}
	r.tick(Input{}, .02)
	if !o.Complete || r.Status != "cleared" || r.Stats.Kills != 0 || len(r.Drops) != 0 {
		t.Fatal("gate completion or reward isolation failed")
	}
}
func TestRuneGateWrongSealResetsWithoutDamage(t *testing.T) {
	r := gateTestRun()
	if r.RoomObjective == nil {
		t.Fatal("gate missing")
	}
	r.Enemies[0].HP = 0
	o := r.RoomObjective
	hp := r.Player.HP
	touchGateRune(r, o.Sequence[0])
	touchGateRune(r, o.Sequence[2])
	if o.Collected != 0 || r.Player.HP != hp {
		t.Fatal("wrong seal did not reset safely")
	}
	for _, p := range o.Pickups {
		if p.Collected {
			t.Fatal("wrong sequence left a rune lit")
		}
	}
}
func TestRuneGateProgressAndStandingLatchPersist(t *testing.T) {
	r := gateTestRun()
	if r.RoomObjective == nil {
		t.Fatal("gate missing")
	}
	r.Enemies[0].HP = 0
	touchGateRune(r, r.RoomObjective.Sequence[0])
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 30; n++ {
		saved.tickRuneGateObjective()
	}
	if saved.RoomObjective.Collected != 1 {
		t.Fatal("standing/reload retriggered rune")
	}
	for _, id := range saved.RoomObjective.Sequence[1:] {
		touchGateRune(&saved, id)
	}
	if !saved.RoomObjective.Complete {
		t.Fatal("saved puzzle could not finish")
	}
}

func TestRuneGateJumpAndPauseDoNotRetriggerSeal(t *testing.T) {
	r := gateTestRun()
	r.Enemies[0].HP = 0
	touchGateRune(r, r.RoomObjective.Sequence[0])
	r.Player.Jump = .5
	r.tickRuneGateObjective()
	r.Player.Jump = 0
	r.tickRuneGateObjective()
	if r.RoomObjective.Collected != 1 {
		t.Fatal("jumping in place retriggered seal")
	}
	r.Paused = true
	touchGateRune(r, r.RoomObjective.Sequence[1])
	if r.RoomObjective.Collected != 1 {
		t.Fatal("paused puzzle advanced")
	}
}

func TestRuneGateSwitchesOffHazardsAfterPatrol(t *testing.T) {
	r := gateTestRun()
	r.Level.Rooms[0].Hazards = []Hazard{{Kind: "fire", Period: 3, Duration: 1}}
	r.tickRuneGateObjective()
	if r.Level.Rooms[0].Hazards[0].Disabled {
		t.Fatal("hazards disabled during combat")
	}
	r.Enemies[0].HP = 0
	r.tickRuneGateObjective()
	if !r.Level.Rooms[0].Hazards[0].Disabled {
		t.Fatal("hazards stayed active during puzzle")
	}
}
func TestRuneGateCampaignPlacement(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			want := level.ID%10 == 9 && room == 0
			if (arena.Objective == "rune_gate") != want {
				t.Fatalf("wrong gate placement %d/%d", level.ID, room)
			}
			if want {
				count++
			}
		}
	}
	if count != 10 {
		t.Fatal("missing rune gate region")
	}
}

func TestRuneSealsReachableAcrossEveryArena(t *testing.T) {
	type cell struct{ x, y int }
	for _, level := range Campaign() {
		r := NewRunAtLevel("gate-layout", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level.ID)
		for room := range level.Rooms {
			r.Room = room
			r.Level.Rooms[room].Objective = "rune_gate"
			r.spawnRoom()
			start := cell{16, 41}
			queue := []cell{start}
			seen := map[cell]bool{start: true}
			for head := 0; head < len(queue); head++ {
				at := queue[head]
				for _, delta := range []cell{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					next := cell{at.x + delta.x, at.y + delta.y}
					if seen[next] || next.x < 4 || next.x > 156 || next.y < 32 || next.y > 49 {
						continue
					}
					blocked := false
					for _, obstacle := range r.Arena().solidObstacles() {
						blocked = blocked || contains(obstacle, float64(next.x*10), float64(next.y*10), 10)
					}
					if !blocked {
						seen[next] = true
						queue = append(queue, next)
					}
				}
			}
			for _, pickup := range r.RoomObjective.Pickups {
				reached := false
				target := Actor{X: pickup.X, Y: pickup.Y}
				for at := range seen {
					position := Actor{X: float64(at.x * 10), Y: float64(at.y * 10)}
					if math.Hypot(position.X-pickup.X, position.Y-pickup.Y) <= 28 && r.clearMeleePath(&position, &target) {
						reached = true
						break
					}
				}
				if !reached {
					t.Fatalf("mission %d tier %d seal %d unreachable", level.ID, room+1, pickup.ID)
				}
			}
		}
	}
}
