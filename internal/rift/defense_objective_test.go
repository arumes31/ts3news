package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
	"ts3news/internal/content"
)

func defenseTestRun() *Run {
	r := circleTestRun()
	r.Level.Rooms[0].Objective = "split_defense"
	r.Enemies = []Actor{{ID: "upper", Kind: "goblin", X: 800, Y: 325, HP: 100, MaxHP: 100, Speed: 90}, {ID: "lower", Kind: "wolf", X: 800, Y: 485, HP: 100, MaxHP: 100, Speed: 90}}
	r.beginRoomObjective()
	return r
}

func TestDefenseAssignsBothLanesAndExcludesSpecialActors(t *testing.T) {
	r := defenseTestRun()
	r.Enemies = append(r.Enemies, Actor{ID: "treasure", Kind: "treasure", HP: 100}, Actor{ID: "boss", Kind: "boss", HP: 100}, Actor{ID: "prop", Kind: "totem", HP: 100})
	r.beginRoomObjective()
	for i, id := range []string{"upper", "lower"} {
		lane := r.RoomObjective.Lanes[i]
		if len(lane.EnemyIDs) != 1 || lane.EnemyIDs[0] != id || lane.Ward.HP != 100 {
			t.Fatalf("unexpected lane: %+v", lane)
		}
	}
}

func TestDefenseEnemiesApproachAssignedWardAndCanBeIntercepted(t *testing.T) {
	r := defenseTestRun()
	for i := range r.Enemies {
		e := &r.Enemies[i]
		if !r.tickDefenseEnemy(e, .5) || e.X >= 800 {
			t.Fatal("enemy did not approach ward")
		}
		if e.Y != r.RoomObjective.Lanes[i].Ward.Y {
			t.Fatal("enemy left assigned lane")
		}
		r.Player.X, r.Player.Y = e.X-50, e.Y
		if r.tickDefenseEnemy(e, .5) {
			t.Fatal("nearby player cannot intercept")
		}
		r.Player.X = 160
	}
}

func TestDefenseDamageIsLaneSpecificAndInterruptible(t *testing.T) {
	r := defenseTestRun()
	o := r.RoomObjective
	r.Enemies[0].X, r.Enemies[0].Y = o.Lanes[0].Ward.X, o.Lanes[0].Ward.Y
	r.tickDefenseObjective(1)
	if o.Lanes[0].Ward.HP != 95 || o.Lanes[1].Ward.HP != 100 || !o.Lanes[0].Contested || o.Lanes[1].Contested {
		t.Fatal("lane damage crossed lanes")
	}
	r.Enemies[0].Knockdown = 1
	r.tickDefenseObjective(1)
	if o.Lanes[0].Ward.HP != 95 || o.Lanes[0].Contested {
		t.Fatal("knockdown did not interrupt ward damage")
	}
	r.Enemies[0].Knockdown = 0
	r.Paused = true
	r.tickDefenseObjective(5)
	if o.Lanes[0].Ward.HP != 95 {
		t.Fatal("paused ward damaged")
	}
}

func TestDefenseReloadRetainsAssignmentsHealthAndPulse(t *testing.T) {
	r := defenseTestRun()
	r.RoomObjective.Lanes[1].Ward.HP = 55
	r.RoomObjective.Lanes[1].Seconds = .75
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	lane := &saved.RoomObjective.Lanes[1]
	if lane.Ward.HP != 55 || lane.Seconds != .75 || lane.EnemyIDs[0] != "lower" {
		t.Fatal("lane state lost")
	}
	saved.Enemies[1].X, saved.Enemies[1].Y = lane.Ward.X, lane.Ward.Y
	saved.tickDefenseObjective(.25)
	if lane.Ward.HP != 50 || lane.Seconds != 0 {
		t.Fatal("saved damage pulse changed")
	}
}

func TestDefenseLostWardEndsRunWithoutDamagingPlayer(t *testing.T) {
	for lane := 0; lane < 2; lane++ {
		r := defenseTestRun()
		ward := &r.RoomObjective.Lanes[lane].Ward
		ward.HP = 1
		r.RoomObjective.Lanes[lane].Seconds = .99
		r.Enemies[lane].X, r.Enemies[lane].Y = ward.X, ward.Y
		r.tick(Input{}, .02)
		if r.Status != "defeated" || ward.HP != 0 || r.Player.HP <= 0 || len(r.Drops) != 0 {
			t.Fatal("ward loss did not safely end run")
		}
	}
}

func TestDefenseVictoryRequiresEntirePatrolAndPreservesNormalRewards(t *testing.T) {
	r := defenseTestRun()
	r.hurtEnemy(0, 1000, "hit")
	r.tick(Input{}, .02)
	if r.RoomObjective.Complete {
		t.Fatal("one lane cleared the whole room")
	}
	r.hurtEnemy(1, 1000, "hit")
	r.tick(Input{}, .02)
	if r.Status != "cleared" || !r.RoomObjective.Complete || r.RoomObjective.Collected != 2 || r.Stats.Kills != 2 || len(r.Drops) != 2 {
		t.Fatal("defense victory changed combat rewards")
	}
}

func TestDefenseApproachesReachWardsInEveryRegion(t *testing.T) {
	for mission := 10; mission <= 100; mission += 10 {
		r := NewRunAtLevel("lane-route", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), mission)
		r.Level.Rooms[0].Objective = "split_defense"
		r.Enemies = []Actor{{ID: "upper", Kind: "goblin", X: 1000, Y: 325, HP: 100, MaxHP: 100, Speed: 90}, {ID: "lower", Kind: "wolf", X: 1000, Y: 485, HP: 100, MaxHP: 100, Speed: 90}}
		r.beginRoomObjective()
		r.Player.X, r.Player.Y = 1550, 410
		for i := range r.Enemies {
			e := &r.Enemies[i]
			ward := r.RoomObjective.Lanes[i].Ward
			for n := 0; n < 3000 && math.Hypot(e.X-ward.X, e.Y-ward.Y) > 45; n++ {
				r.tickDefenseEnemy(e, .02)
			}
			if math.Hypot(e.X-ward.X, e.Y-ward.Y) > 45 || !r.clearMeleePath(e, &ward) {
				t.Fatalf("mission %d lane %d stuck at %.1f/%.1f", mission, i, e.X, e.Y)
			}
		}
	}
}

func TestDefenseCampaignPlacement(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			want := level.ID%10 == 0 && room == 0
			if (arena.Objective == "split_defense") != want {
				t.Fatalf("wrong defense placement %d/%d", level.ID, room)
			}
			if want {
				count++
			}
		}
	}
	if count != 10 {
		t.Fatal("missing regional defense")
	}
}

func TestDefenseEveryAssignedSpawnCanReachItsWard(t *testing.T) {
	for mission := 10; mission <= 100; mission += 10 {
		r := NewRunAtLevel("patrol-lanes", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), mission)
		r.Player.X, r.Player.Y = 35, 410
		for _, lane := range r.RoomObjective.Lanes {
			for _, id := range lane.EnemyIDs {
				for i := range r.Enemies {
					e := &r.Enemies[i]
					if e.ID != id {
						continue
					}
					for n := 0; n < 6000 && (math.Hypot(e.X-lane.Ward.X, e.Y-lane.Ward.Y) > 45 || !r.clearMeleePath(e, &lane.Ward)); n++ {
						r.tickDefenseEnemy(e, .02)
					}
					if math.Hypot(e.X-lane.Ward.X, e.Y-lane.Ward.Y) > 45 || !r.clearMeleePath(e, &lane.Ward) {
						t.Fatalf("mission %d enemy %s stuck at %.1f/%.1f", mission, e.ID, e.X, e.Y)
					}
				}
			}
		}
	}
}
