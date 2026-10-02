package rift

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
	"ts3news/internal/content"
)

func totemTestRun() *Run {
	r := circleTestRun()
	r.Level.Rooms[0].Objective = "destroy_totems"
	r.beginRoomObjective()
	return r
}
func TestTotemsAreStationaryDamageableProps(t *testing.T) {
	r := totemTestRun()
	if len(r.Enemies) != 4 {
		t.Fatal("totems missing")
	}
	target := r.Enemies[1]
	for n := 0; n < 100; n++ {
		r.enemyTick(1, .02)
	}
	if r.Enemies[1].X != target.X || r.Enemies[1].Y != target.Y || r.Enemies[1].Windup != 0 {
		t.Fatal("totem moved or attacked")
	}
	r.hurtEnemy(1, target.MaxHP/2, "hit")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Enemies[1].HP != target.MaxHP/2 {
		t.Fatal("totem damage lost on reload")
	}
	saved.hurtEnemy(1, target.MaxHP, "hit")
	saved.hurtEnemy(1, target.MaxHP, "hit")
	if saved.RoomObjective.Collected != 1 || saved.Stats.Kills != 0 || len(saved.Drops) != 0 {
		t.Fatal("prop destruction granted monster rewards or double counted")
	}
	for i := 2; i < len(saved.Enemies); i++ {
		saved.hurtEnemy(i, saved.Enemies[i].MaxHP, "fire")
	}
	saved.tick(Input{}, .02)
	if !saved.RoomObjective.Complete || saved.Status != "fighting" {
		t.Fatal("living patrol did not gate room clear")
	}
	saved.hurtEnemy(0, saved.Enemies[0].MaxHP, "hit")
	saved.tick(Input{}, .02)
	if saved.Status != "cleared" || saved.Stats.Kills != 1 || len(saved.Drops) != 1 {
		t.Fatal("patrol rewards or final clear incorrect")
	}
}
func TestTotemsDoNotGrantAerialMonsterFinish(t *testing.T) {
	r := totemTestRun()
	r.Enemies[1].X, r.Enemies[1].Y, r.Enemies[1].HP = r.Player.X+30, r.Player.Y, 1
	r.Player.Jump = .5
	r.tick(Input{Attack: true}, .02)
	if r.Enemies[1].HP != 0 || r.Stats.AerialFinishes != 0 {
		t.Fatal("totem credited as aerial monster finish")
	}
}
func TestLivingTotemsBlockPatrolClear(t *testing.T) {
	r := totemTestRun()
	r.Enemies[0].HP = 0
	r.tick(Input{}, .02)
	if r.Status != "fighting" || r.RoomObjective.Complete {
		t.Fatal("living totems bypassed")
	}
}

func TestTotemPlacementsReachableAcrossEveryArena(t *testing.T) {
	type cell struct{ x, y int }
	for _, level := range Campaign() {
		r := NewRunAtLevel("totem-layout", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level.ID)
		for room := range level.Rooms {
			r.Room = room
			r.Level.Rooms[room].Objective = "destroy_totems"
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
				if pickup.Kind != "totem" {
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
					t.Fatalf("mission %d tier %d totem %s unreachable", level.ID, room+1, pickup.ID)
				}
			}
		}
	}
}

func TestTotemCampaignDistribution(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			expected := level.ID%10 == 6 && room == 1
			if (arena.Objective == "destroy_totems") != expected {
				t.Fatal("wrong placement")
			}
			if expected {
				count++
				if !strings.Contains(level.Tactic, "destroy three ritual totems") {
					t.Fatal("preview missing")
				}
			}
		}
	}
	if count != 10 {
		t.Fatal("missing region")
	}
}
