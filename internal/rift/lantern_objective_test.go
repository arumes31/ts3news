package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
	"ts3news/internal/content"
)

func lanternTestRun() *Run {
	r := circleTestRun()
	r.Level.Rooms[0].Objective = "protect_lantern"
	r.beginRoomObjective()
	return r
}
func TestLanternNearbyThreatsDrainAndDistanceProtects(t *testing.T) {
	r := lanternTestRun()
	if r.RoomObjective == nil {
		t.Fatal("lantern missing")
	}
	o := r.RoomObjective
	r.Enemies[0].X, r.Enemies[0].Y = o.Lantern.X, o.Lantern.Y
	r.tickLanternObjective(1)
	if o.Lantern.HP != 95 || !o.Contested {
		t.Fatal("nearby enemy did not drain lantern")
	}
	r.Enemies[0].X = 1500
	r.tickLanternObjective(1)
	if o.Lantern.HP != 95 || o.Contested {
		t.Fatal("distant enemy drained lantern")
	}
	r.Enemies[0].X = o.Lantern.X
	r.Paused = true
	r.tickLanternObjective(2)
	if o.Lantern.HP != 95 {
		t.Fatal("paused lantern drained")
	}
}
func TestLanternProgressPersistsAndClearProtects(t *testing.T) {
	r := lanternTestRun()
	if r.RoomObjective == nil {
		t.Fatal("lantern missing")
	}
	r.RoomObjective.Lantern.HP = 47
	r.RoomObjective.Seconds = .4
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.RoomObjective.Lantern.HP != 47 || saved.RoomObjective.Seconds != .4 {
		t.Fatal("lantern save lost")
	}
	saved.hurtEnemy(0, 1000, "hit")
	saved.tick(Input{}, .02)
	if saved.Status != "cleared" || !saved.RoomObjective.Complete || saved.Stats.Kills != 1 || len(saved.Drops) != 1 {
		t.Fatal("lantern success or patrol rewards incorrect")
	}
}
func TestLanternExtinctionFailsExpeditionImmediately(t *testing.T) {
	r := lanternTestRun()
	if r.RoomObjective == nil {
		t.Fatal("lantern missing")
	}
	r.RoomObjective.Lantern.HP = 1
	r.RoomObjective.Seconds = .99
	r.Enemies[0].X, r.Enemies[0].Y = r.RoomObjective.Lantern.X, r.RoomObjective.Lantern.Y
	r.tick(Input{}, .02)
	if r.Status != "defeated" || r.RoomObjective.Lantern.HP != 0 || r.RoomObjective.Complete {
		t.Fatal("extinguished lantern did not fail expedition")
	}
	if r.Player.HP <= 0 {
		t.Fatal("lantern failure incorrectly damaged player")
	}
}

func TestLanternCapsThreatDrainAndIgnoresProps(t *testing.T) {
	r := lanternTestRun()
	l := r.RoomObjective.Lantern
	r.Enemies = nil
	for i := 0; i < 5; i++ {
		r.Enemies = append(r.Enemies, Actor{Kind: "goblin", X: l.X, Y: l.Y, HP: 100})
	}
	r.tickLanternObjective(1)
	if l.HP != 85 {
		t.Fatal("drain exceeded three-enemy cap")
	}
	for i := range r.Enemies {
		r.Enemies[i].Kind = "cage"
	}
	r.tickLanternObjective(1)
	if l.HP != 85 {
		t.Fatal("objective props drained lantern")
	}
}

func TestLanternReachableAcrossEveryArena(t *testing.T) {
	type cell struct{ x, y int }
	for _, level := range Campaign() {
		r := NewRunAtLevel("lantern-layout", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level.ID)
		for room := range level.Rooms {
			r.Room = room
			r.Level.Rooms[room].Objective = "protect_lantern"
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
			for _, pickup := range []Actor{*r.RoomObjective.Lantern} {
				if pickup.Kind != "lantern" {
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
					t.Fatalf("mission %d tier %d lantern %s unreachable", level.ID, room+1, pickup.ID)
				}
			}
		}
	}
}

func TestLanternCampaignPlacement(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			want := level.ID%10 == 8 && room == 0
			if (arena.Objective == "protect_lantern") != want {
				t.Fatalf("wrong lantern placement %d/%d", level.ID, room)
			}
			if want {
				count++
			}
		}
	}
	if count != 10 {
		t.Fatal("missing lantern region")
	}
}
