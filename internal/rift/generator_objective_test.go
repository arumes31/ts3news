package rift

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
	"ts3news/internal/content"
)

func generatorTestRun() *Run {
	r := circleTestRun()
	r.Level.Rooms[0].Objective = "disable_generators"
	r.Level.Rooms[0].Hazards = []Hazard{
		{Obstacle: Obstacle{X: 180, Y: 390, W: 80, H: 50}, Kind: "fire", Period: 7, Duration: 1},
		{Obstacle: Obstacle{X: 680, Y: 390, W: 80, H: 50}, Kind: "ice", Period: 7, Duration: 1},
	}
	r.beginRoomObjective()
	return r
}
func TestGeneratorDisablesOnlyItsLinkedHazardAndPersists(t *testing.T) {
	r := generatorTestRun()
	if r.RoomObjective == nil || r.RoomObjective.Target != 2 {
		t.Fatal("generators missing")
	}
	r.Player.X, r.Player.Y, r.Clock = 200, 410, 1.3
	before := r.Player.HP
	r.hazardTick()
	if r.Player.HP >= before {
		t.Fatal("powered hazard did not fire")
	}
	r.hurtEnemy(1, r.Enemies[1].MaxHP, "fire")
	if !r.Level.Rooms[0].Hazards[0].Disabled || r.Level.Rooms[0].Hazards[1].Disabled {
		t.Fatal("wrong hazard disabled")
	}
	if r.RoomObjective.Collected != 1 || r.RoomObjective.Complete || r.Stats.Kills != 0 || len(r.Drops) != 0 {
		t.Fatal("shutdown counted incorrectly")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	saved.SkillTimers = map[string]float64{}
	before = saved.Player.HP
	saved.hazardTick()
	if saved.Player.HP != before {
		t.Fatal("saved disabled hazard still damages")
	}
	saved.Player.X = 700
	saved.hazardTick()
	if saved.Player.HP >= before {
		t.Fatal("unlinked hazard was disabled")
	}
	saved.hurtEnemy(2, saved.Enemies[2].MaxHP, "hit")
	saved.tick(Input{}, .02)
	if !saved.RoomObjective.Complete || saved.Status != "fighting" {
		t.Fatal("patrol gate missing")
	}
	saved.Enemies[0].HP = 0
	saved.tick(Input{}, .02)
	if saved.Status != "cleared" {
		t.Fatal("shutdown room did not clear")
	}
}

func TestGeneratorPlacementsReachableAcrossEveryArena(t *testing.T) {
	type cell struct{ x, y int }
	for _, level := range Campaign() {
		r := NewRunAtLevel("generator-layout", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level.ID)
		for room := range level.Rooms {
			r.Room = room
			r.Level.Rooms[room].Objective = "disable_generators"
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
			for _, pickup := range r.Enemies {
				if pickup.Kind != "generator" {
					continue
				}
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
					t.Fatalf("mission %d tier %d generator %s unreachable", level.ID, room+1, pickup.ID)
				}
			}
		}
	}
}

func TestGeneratorCampaignPlacement(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			expected := level.ID%10 == 8 && room == 1
			if (arena.Objective == "disable_generators") != expected {
				t.Fatal("wrong generator placement")
			}
			if expected {
				count++
				if !strings.Contains(level.Tactic, "shut down linked floor hazards") {
					t.Fatal("missing preview")
				}
			}
		}
	}
	if count != 10 {
		t.Fatal("missing generator region")
	}
}
