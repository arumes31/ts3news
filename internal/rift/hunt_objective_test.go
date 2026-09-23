package rift

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"ts3news/internal/content"
)

func huntTestRun() *Run {
	r := circleTestRun()
	r.Level.Rooms[0].Objective = "marked_hunt"
	r.Enemies = []Actor{
		{ID: "a", Kind: "goblin", HP: 100, MaxHP: 100, X: 700, Y: 410, Knockdown: 100},
		{ID: "b", Kind: "archer", HP: 100, MaxHP: 100, X: 800, Y: 410, Knockdown: 100},
		{ID: "c", Kind: "knight", HP: 100, MaxHP: 100, X: 900, Y: 410, Knockdown: 100},
		{ID: "d", Kind: "goblin", HP: 100, MaxHP: 100, X: 1000, Y: 410, Knockdown: 100},
		{ID: "e", Kind: "treasure", HP: 100, MaxHP: 100, X: 1100, Y: 410, Knockdown: 100},
	}
	r.beginRoomObjective()
	return r
}
func TestHuntTargetsPersistAndSurvivorsWithdrawWithoutRewards(t *testing.T) {
	r := huntTestRun()
	o := r.RoomObjective
	if o == nil || len(o.Targets) != 3 {
		t.Fatal("hunt targets missing")
	}
	for _, id := range o.Targets {
		if id == "e" {
			t.Fatal("fleeing treasure selected as required target")
		}
	}
	first := o.Targets[0]
	for i := range r.Enemies {
		if r.Enemies[i].ID == first {
			r.hurtEnemy(i, 1000, "hit")
		}
	}
	r.tick(Input{}, .02)
	if o.Collected != 1 || o.Complete {
		t.Fatal("partial hunt incorrectly completed")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.RoomObjective.Targets[0] != first || saved.RoomObjective.Collected != 1 {
		t.Fatal("saved hunt changed")
	}
	for _, id := range saved.RoomObjective.Targets {
		for i := range saved.Enemies {
			if saved.Enemies[i].ID == id {
				saved.hurtEnemy(i, 1000, "fire")
			}
		}
	}
	saved.tick(Input{}, .02)
	if saved.Status != "cleared" || !saved.RoomObjective.Complete {
		t.Fatal("hunt did not clear")
	}
	if saved.Stats.Kills != 3 || len(saved.Drops) != 3 || saved.Stats.TreasureGoblins != 0 {
		t.Fatal("withdrawal granted rewards")
	}
	escaped := 0
	for _, e := range saved.Enemies {
		if e.Pose == "escape" {
			escaped++
		}
	}
	if escaped != 2 {
		t.Fatal("surviving patrol did not withdraw")
	}
}
func TestHuntDoesNotCountUnmarkedDefeats(t *testing.T) {
	r := huntTestRun()
	if r.RoomObjective == nil {
		t.Fatal("hunt missing")
	}
	r.hurtEnemy(4, 1000, "hit")
	r.tick(Input{}, .02)
	if r.RoomObjective.Collected != 0 || r.Status != "fighting" {
		t.Fatal("unmarked death counted")
	}
}

func TestHuntDoesNotResolveWhilePausedOrDead(t *testing.T) {
	for _, mode := range []string{"paused", "dead"} {
		t.Run(mode, func(t *testing.T) {
			r := huntTestRun()
			for _, id := range r.RoomObjective.Targets {
				for i := range r.Enemies {
					if r.Enemies[i].ID == id {
						r.Enemies[i].HP = 0
					}
				}
			}
			if mode == "paused" {
				r.Paused = true
			} else {
				r.Player.HP = 0
			}
			r.tickHuntObjective()
			if r.RoomObjective.Complete || r.Enemies[4].HP == 0 {
				t.Fatal("inactive hunt resolved")
			}
		})
	}
}
func TestHuntSkipsRosterWithoutHuntableEnemies(t *testing.T) {
	r := huntTestRun()
	r.Enemies = []Actor{{ID: "treasure", Kind: "treasure", HP: 100, MaxHP: 100}}
	r.beginRoomObjective()
	if r.RoomObjective != nil {
		t.Fatal("unwinnable mandatory hunt")
	}
}

func TestHuntCampaignTargetsUseSharedRoster(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			expected := level.ID%10 == 9 && room == 1
			if (arena.Objective == "marked_hunt") != expected {
				t.Fatal("wrong hunt placement")
			}
			if !expected {
				continue
			}
			count++
			if !strings.Contains(level.Tactic, "survivors retreat without loot") {
				t.Fatal("missing preview")
			}
			r := NewRunAtLevel("hunt-campaign", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level.ID)
			r.Room = room
			r.spawnRoom()
			if r.RoomObjective == nil || len(r.Enemies) != len(r.EncounterPlan[room]) {
				t.Fatal("hunt altered roster")
			}
			for _, id := range r.RoomObjective.Targets {
				found := false
				for _, e := range r.EncounterPlan[room] {
					if e.ID == id && e.Kind != "treasure" {
						found = true
					}
				}
				if !found {
					t.Fatal("target not from eligible shared roster")
				}
			}
		}
	}
	if count != 10 {
		t.Fatal("missing hunt region")
	}
}
