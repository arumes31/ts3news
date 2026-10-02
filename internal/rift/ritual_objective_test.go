package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
	"ts3news/internal/content"
)

func ritualTestRun() *Run {
	r := circleTestRun()
	r.Level.Rooms[0].Objective = "interrupt_ritual"
	r.Enemies = []Actor{
		{ID: "a", Kind: "goblin", X: 400, Y: 330, HP: 100, MaxHP: 100, Damage: 20},
		{ID: "b", Kind: "archer", X: 800, Y: 480, HP: 100, MaxHP: 100, Damage: 20},
		{ID: "c", Kind: "goblin", X: 1200, Y: 330, HP: 100, MaxHP: 100, Damage: 20},
		{ID: "patrol", Kind: "goblin", X: 1400, Y: 490, HP: 100, MaxHP: 100, Knockdown: 100},
	}
	r.beginRoomObjective()
	return r
}
func TestRitualDamageInterruptsAndProgressPersists(t *testing.T) {
	r := ritualTestRun()
	if r.RoomObjective == nil {
		t.Fatal("ritual missing")
	}
	r.enemyTick(0, 3)
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.RoomObjective.Channels[0].Seconds != 3 {
		t.Fatal("channel progress lost")
	}
	saved.hurtEnemy(0, 1, "hit")
	if saved.RoomObjective.Channels[0].Seconds != 0 {
		t.Fatal("damage did not interrupt")
	}
	if saved.Enemies[0].X != 400 || saved.Enemies[0].Y != 330 {
		t.Fatal("channeler moved")
	}
}
func TestRitualPulsesAndPatrolGate(t *testing.T) {
	r := ritualTestRun()
	if r.RoomObjective == nil {
		t.Fatal("ritual missing")
	}
	r.Player.X, r.Player.Y = 410, 330
	hp := r.Player.HP
	r.enemyTick(0, 8)
	if r.Player.HP >= hp || r.RoomObjective.Channels[0].Seconds != 0 {
		t.Fatal("ritual pulse missing")
	}
	for i := 0; i < 3; i++ {
		r.hurtEnemy(i, 1000, "hit")
	}
	r.tick(Input{}, .02)
	if !r.RoomObjective.Complete || r.Status != "fighting" || r.Stats.Kills != 3 {
		t.Fatal("ritual completion or patrol gate incorrect")
	}
	r.hurtEnemy(3, 1000, "hit")
	r.tick(Input{}, .02)
	if r.Status != "cleared" || r.Stats.Kills != 4 || len(r.Drops) != 4 {
		t.Fatal("ritual rewards incorrect")
	}
}
func TestRitualInactiveAndDistantSafety(t *testing.T) {
	for _, mode := range []string{"paused", "dead", "distant"} {
		t.Run(mode, func(t *testing.T) {
			r := ritualTestRun()
			if r.RoomObjective == nil {
				t.Fatal("ritual missing")
			}
			if mode == "paused" {
				r.Paused = true
			}
			if mode == "dead" {
				r.Player.HP = 0
			}
			hp := r.Player.HP
			r.enemyTick(0, 8)
			if r.Player.HP != hp {
				t.Fatal("inactive or distant player damaged")
			}
			if mode != "distant" && r.RoomObjective.Channels[0].Seconds != 0 {
				t.Fatal("inactive ritual advanced")
			}
		})
	}
}

func TestRitualCampaignPlacement(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			want := level.ID > 1 && level.ID%10 == 1 && room == 1
			if (arena.Objective == "interrupt_ritual") != want {
				t.Fatalf("wrong ritual placement in %d/%d", level.ID, room)
			}
			if want {
				count++
			}
		}
	}
	if count != 9 {
		t.Fatal("missing ritual mission")
	}
}
func TestRitualZeroDamageDoesNotInterrupt(t *testing.T) {
	r := ritualTestRun()
	r.enemyTick(0, 3)
	r.hurtEnemy(0, 0, "hit")
	if r.RoomObjective.Channels[0].Seconds != 3 {
		t.Fatal("zero damage reset charge")
	}
}

func TestRitualChannelersReachableInCampaign(t *testing.T) {
	type cell struct{ x, y int }
	for _, level := range Campaign() {
		r := NewRunAtLevel("ritual-layout", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level.ID)
		for room := range level.Rooms {
			r.Room = room
			if r.Level.Rooms[room].Objective != "interrupt_ritual" {
				continue
			}
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
				isChannel := false
				for _, c := range r.RoomObjective.Channels {
					isChannel = isChannel || c.EnemyID == pickup.ID
				}
				if !isChannel {
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
					t.Fatalf("mission %d tier %d channeler %s unreachable", level.ID, room+1, pickup.ID)
				}
			}
		}
	}
}
