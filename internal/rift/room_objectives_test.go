package rift

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
	"ts3news/internal/content"
)

func sigilTestRun() *Run {
	r := NewRunAtLevel("sigils", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
	r.Level.Rooms[0].Objective = "sigils"
	r.Level.Rooms[0].Hazards = nil
	r.spawnRoom()
	return r
}
func TestSigilsGateClearAndPersist(t *testing.T) {
	r := sigilTestRun()
	for i := range r.Enemies {
		r.Enemies[i].HP = 0
	}
	r.tick(Input{}, .02)
	if r.Status != "fighting" || r.RoomObjective == nil || r.RoomObjective.Collected != 0 {
		t.Fatal("room cleared before sigils")
	}
	first := r.RoomObjective.Pickups[0]
	r.Player.X = first.X
	r.Player.Y = first.Y
	r.tick(Input{}, .02)
	r.tick(Input{}, .02)
	if r.RoomObjective.Collected != 1 {
		t.Fatal("pickup replay double-counted")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	for _, pickup := range saved.RoomObjective.Pickups {
		saved.Player.X = pickup.X
		saved.Player.Y = pickup.Y
		saved.tick(Input{}, .02)
	}
	if saved.Status != "cleared" || !saved.RoomObjective.Complete || saved.RoomObjective.Collected != 3 {
		t.Fatal("collected room did not clear")
	}
	saved.FinishCheckpoint("advance", content.AbyssMobCatalog())
	if saved.RoomObjective != nil {
		t.Fatal("sigils leaked into next room")
	}
}
func TestSigilsRequireLiveGroundedUnpausedPlayerAndEnemyClear(t *testing.T) {
	for _, mode := range []string{"paused", "airborne", "defeated", "alive"} {
		t.Run(mode, func(t *testing.T) {
			r := sigilTestRun()
			for i := range r.Enemies {
				r.Enemies[i].Knockdown = 100
			}
			pickup := r.RoomObjective.Pickups[0]
			r.Player.X = pickup.X
			r.Player.Y = pickup.Y
			switch mode {
			case "paused":
				r.SetPaused(true, time.Unix(100, 0))
			case "airborne":
				r.Player.Jump = .5
			case "defeated":
				r.Player.HP = 0
			}
			r.Step(Input{}, time.Unix(100, 0).Add(20*time.Millisecond))
			want := 0
			if mode == "alive" {
				want = 1
			}
			if r.RoomObjective.Collected != want {
				t.Fatal("wrong pickup eligibility")
			}
			if mode == "alive" {
				for _, pickup := range r.RoomObjective.Pickups {
					r.Player.X = pickup.X
					r.Player.Y = pickup.Y
					r.tick(Input{}, .02)
				}
				if r.Status != "fighting" || !r.RoomObjective.Complete {
					t.Fatal("sigils bypassed living enemies")
				}
			}
		})
	}
}

func TestSigilPlacementsReachableAcrossEveryArena(t *testing.T) {
	type cell struct{ x, y int }
	for _, level := range Campaign() {
		r := NewRunAtLevel("sigil-layout", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level.ID)
		for room := range level.Rooms {
			r.Room = room
			r.Level.Rooms[room].Objective = "sigils"
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
					t.Fatalf("mission %d tier %d sigil %d unreachable", level.ID, room+1, pickup.ID)
				}
			}
		}
	}
}

func TestCampaignSigilRoomsAreAnnouncedAndDistributed(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			expected := level.ID%10 == 3 && room == 1
			if (arena.Objective == "sigils") != expected {
				t.Fatalf("unexpected room objective in mission %d tier %d", level.ID, room+1)
			}
			if expected {
				count++
				if !strings.Contains(level.Tactic, "Tier 2: gather three sigils") {
					t.Fatal("missing mission preview")
				}
			}
		}
	}
	if count != 10 {
		t.Fatalf("got %d sigil rooms", count)
	}
}
