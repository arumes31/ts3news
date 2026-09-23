package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
	"ts3news/internal/content"
)

func rescueTestRun() *Run {
	r := circleTestRun()
	r.Level.Rooms[0].Objective = "rescue_companions"
	r.beginRoomObjective()
	return r
}
func TestRescueCagesReleaseWithoutMonsterRewards(t *testing.T) {
	r := rescueTestRun()
	if r.RoomObjective == nil {
		t.Fatal("rescue missing")
	}
	if len(r.RoomObjective.Captives) != 2 || len(r.Enemies) != 3 {
		t.Fatal("captives or cages missing")
	}
	r.hurtEnemy(1, 1000, "hit")
	r.hurtEnemy(1, 1000, "hit")
	if r.RoomObjective.Collected != 1 || !r.RoomObjective.Captives[0].Freed || r.Stats.Kills != 0 || len(r.Drops) != 0 {
		t.Fatal("rescue count or reward isolation incorrect")
	}
	r.hurtEnemy(2, 1000, "fire")
	r.tick(Input{}, .02)
	if !r.RoomObjective.Complete || r.Status != "fighting" {
		t.Fatal("patrol gate missing")
	}
	r.hurtEnemy(0, 1000, "hit")
	r.tick(Input{}, .02)
	if r.Status != "cleared" || r.Stats.Kills != 1 || len(r.Drops) != 1 {
		t.Fatal("patrol reward or clear incorrect")
	}
}
func TestRescueDamageAndReleasePersist(t *testing.T) {
	r := rescueTestRun()
	if r.RoomObjective == nil {
		t.Fatal("rescue missing")
	}
	r.Clock = 2
	r.hurtEnemy(1, 1000, "hit")
	r.hurtEnemy(2, 5, "hit")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if !saved.RoomObjective.Captives[0].Freed || saved.RoomObjective.Captives[0].FreedAt != 2 || saved.Enemies[2].HP != r.Enemies[2].HP {
		t.Fatal("saved rescue progress lost")
	}
	saved.hurtEnemy(2, 1000, "hit")
	if saved.RoomObjective.Collected != 2 || saved.Stats.Kills != 0 {
		t.Fatal("saved rescue did not complete cleanly")
	}
}

func TestRescueCagesStayStillAndDoNotCountAsAerialKills(t *testing.T) {
	r := rescueTestRun()
	cage := r.Enemies[1]
	for n := 0; n < 100; n++ {
		r.enemyTick(1, .02)
	}
	if r.Enemies[1].X != cage.X || r.Enemies[1].Y != cage.Y || r.Enemies[1].Windup != 0 {
		t.Fatal("cage moved or attacked")
	}
	r.Enemies[1].X, r.Enemies[1].Y, r.Enemies[1].HP = r.Player.X+30, r.Player.Y, 1
	r.Player.Jump = .5
	r.tick(Input{Attack: true}, .02)
	if r.Enemies[1].HP != 0 || r.Stats.AerialFinishes != 0 || r.Stats.Kills != 0 {
		t.Fatal("cage credited as monster kill")
	}
}

func TestRescueCagesResistDisplacement(t *testing.T) {
	r := rescueTestRun()
	cage := &r.Enemies[1]
	x, y := cage.X, cage.Y
	r.knockbackActor(cage, 100, 40)
	if cage.X != x || cage.Y != y {
		t.Fatal("cage displaced from captive")
	}
}

func TestRescueCagesReachableAcrossEveryArena(t *testing.T) {
	type cell struct{ x, y int }
	for _, level := range Campaign() {
		r := NewRunAtLevel("rescue-layout", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level.ID)
		for room := range level.Rooms {
			r.Room = room
			r.Level.Rooms[room].Objective = "rescue_companions"
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
				if pickup.Kind != "cage" {
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
					t.Fatalf("mission %d tier %d cage %s unreachable", level.ID, room+1, pickup.ID)
				}
			}
		}
	}
}

func TestRescueCampaignPlacement(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			want := level.ID%10 == 7 && room == 0
			if (arena.Objective == "rescue_companions") != want {
				t.Fatalf("wrong rescue placement %d/%d", level.ID, room)
			}
			if want {
				count++
			}
		}
	}
	if count != 10 {
		t.Fatal("missing rescue region")
	}
}
