package rift

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
	"ts3news/internal/content"
)

func relicTestRun() *Run {
	r := circleTestRun()
	r.Level.Rooms[0].Objective = "carry_relic"
	r.beginRoomObjective()
	return r
}
func TestRelicPickupDeliveryAndPersistence(t *testing.T) {
	r := relicTestRun()
	if r.RoomObjective == nil || r.RoomObjective.Relic == nil {
		t.Fatal("relic missing")
	}
	o := r.RoomObjective
	r.Player.X, r.Player.Y = o.Zone.X, o.Zone.Y
	r.tick(Input{}, .02)
	if o.Complete {
		t.Fatal("delivery without pickup")
	}
	r.Player.X, r.Player.Y = o.Relic.X, o.Relic.Y
	r.tick(Input{}, .02)
	if !o.Carrying || !o.Relic.Collected || o.Complete {
		t.Fatal("pickup state incorrect")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if !saved.RoomObjective.Carrying {
		t.Fatal("carried relic lost on reload")
	}
	saved.Player.X, saved.Player.Y = o.Zone.X, o.Zone.Y
	saved.tick(Input{}, .02)
	if saved.RoomObjective.Carrying || !saved.RoomObjective.Complete || saved.Status != "fighting" {
		t.Fatal("delivery or patrol gate incorrect")
	}
	saved.Enemies[0].HP = 0
	saved.tick(Input{}, .02)
	if saved.Status != "cleared" {
		t.Fatal("patrol defeat did not secure delivery")
	}
}
func TestRelicPickupRequiresLivingGroundedActivePlayer(t *testing.T) {
	for _, mode := range []string{"paused", "dead", "airborne"} {
		t.Run(mode, func(t *testing.T) {
			r := relicTestRun()
			if r.RoomObjective == nil {
				t.Fatal("relic missing")
			}
			r.Player.X, r.Player.Y = r.RoomObjective.Relic.X, r.RoomObjective.Relic.Y
			switch mode {
			case "paused":
				r.Paused = true
			case "dead":
				r.Player.HP = 0
			case "airborne":
				r.Player.Jump = .5
			}
			r.tickRelicObjective()
			if r.RoomObjective.Carrying {
				t.Fatal("invalid pickup")
			}
		})
	}
}
func TestRelicSpeedPenaltyComposesWithGuardAndSlow(t *testing.T) {
	for _, mode := range []string{"walk", "guard", "slowed", "jump"} {
		t.Run(mode, func(t *testing.T) {
			measure := func(carry bool) float64 {
				r := relicTestRun()
				if r.RoomObjective == nil {
					t.Fatal("relic missing")
				}
				r.RoomObjective.Carrying = carry
				r.RoomObjective.Relic.Collected = carry
				if mode == "slowed" {
					r.SkillTimers["slowed"] = 2
				}
				if mode == "jump" {
					r.Player.Jump = .5
				}
				start := r.Player.X
				r.tick(Input{X: 1, Guard: mode == "guard"}, .02)
				return r.Player.X - start
			}
			normal, carried := measure(false), measure(true)
			if normal <= 0 || math.Abs(carried/normal-.7) > 1e-8 {
				t.Fatal("carry speed is not seventy percent", normal, carried)
			}
		})
	}
}

func TestRelicEndpointsReachableAcrossEveryArena(t *testing.T) {
	type cell struct{ x, y int }
	for _, level := range Campaign() {
		r := NewRunAtLevel("relic-layout", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level.ID)
		for room := range level.Rooms {
			r.Room = room
			r.Level.Rooms[room].Objective = "carry_relic"
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
			for _, pickup := range []ObjectivePickup{*r.RoomObjective.Relic, {ID: 2, X: r.RoomObjective.Zone.X, Y: r.RoomObjective.Zone.Y}} {
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
					t.Fatalf("mission %d tier %d relic endpoint %d unreachable", level.ID, room+1, pickup.ID)
				}
			}
		}
	}
}

func TestRelicCampaignPlacement(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			expected := level.ID%10 == 7 && room == 1
			if (arena.Objective == "carry_relic") != expected {
				t.Fatal("wrong relic placement")
			}
			if expected {
				count++
				if !strings.Contains(level.Tactic, "30% slower") {
					t.Fatal("preview must disclose slowdown")
				}
			}
		}
	}
	if count != 10 {
		t.Fatal("missing relic region")
	}
}
func TestRelicDeliveryRestoresSpeed(t *testing.T) {
	r := relicTestRun()
	o := r.RoomObjective
	o.Carrying = true
	o.Relic.Collected = true
	r.Player.X, r.Player.Y = o.Zone.X, o.Zone.Y
	r.tick(Input{}, .02)
	before := r.Player.X
	r.tick(Input{X: -1}, .02)
	if math.Abs(before-r.Player.X-235*.02) > 1e-8 {
		t.Fatal("delivery retained slowdown")
	}
}
