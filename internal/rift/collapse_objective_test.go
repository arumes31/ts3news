package rift

import (
	"encoding/json"
	"testing"
)

func collapseTestRun() *Run {
	r := circleTestRun()
	r.Level.Rooms[0].Objective = "escape_collapse"
	r.beginRoomObjective()
	return r
}
func TestCollapseWarningAndSavedAdvance(t *testing.T) {
	r := collapseTestRun()
	if r.RoomObjective == nil {
		t.Fatal("collapse missing")
	}
	r.tickCollapseObjective(2)
	if r.RoomObjective.CollapseX != 0 {
		t.Fatal("collapse ignored warning")
	}
	r.tickCollapseObjective(2)
	if r.RoomObjective.CollapseX != 90 {
		t.Fatal("collapse speed incorrect")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	saved.tickCollapseObjective(1)
	if saved.RoomObjective.CollapseX != 180 {
		t.Fatal("collapse progress lost")
	}
}
func TestCollapseDamagesBehindFrontAndPauses(t *testing.T) {
	r := collapseTestRun()
	if r.RoomObjective == nil {
		t.Fatal("collapse missing")
	}
	r.Player.X = 50
	hp := r.Player.HP
	r.tickCollapseObjective(4)
	if r.Player.HP >= hp {
		t.Fatal("collapse did not damage caught player")
	}
	hp = r.Player.HP
	r.Paused = true
	x := r.RoomObjective.CollapseX
	r.tickCollapseObjective(2)
	if r.Player.HP != hp || r.RoomObjective.CollapseX != x {
		t.Fatal("paused collapse advanced")
	}
	r.Paused = false
	r.Player.X = 1500
	r.tickCollapseObjective(.5)
	if r.Player.HP != hp {
		t.Fatal("safe player damaged")
	}
}
func TestCollapseExitSecuresWithoutSurvivorRewards(t *testing.T) {
	r := collapseTestRun()
	if r.RoomObjective == nil {
		t.Fatal("collapse missing")
	}
	r.Player.X, r.Player.Y = r.RoomObjective.Zone.X, r.RoomObjective.Zone.Y
	r.tick(Input{}, .02)
	if !r.RoomObjective.Complete || r.Status != "cleared" {
		t.Fatal("exit did not secure room")
	}
	if r.Stats.Kills != 0 || len(r.Drops) != 0 || r.Enemies[0].Pose != "escape" {
		t.Fatal("survivor granted rewards")
	}
}
func TestCollapseLethalDamageRecordsDefeatImmediately(t *testing.T) {
	r := collapseTestRun()
	if r.RoomObjective == nil {
		t.Fatal("collapse missing")
	}
	r.Player.X = 50
	r.Player.HP = 1
	r.RoomObjective.Seconds = 4
	r.tick(Input{}, .02)
	if r.Status != "defeated" || r.RoomObjective.Complete {
		t.Fatal("lethal collapse did not defeat player")
	}
}

func TestCollapseDamageCadenceAndAirborneExit(t *testing.T) {
	r := collapseTestRun()
	r.Player.X = 50
	r.RoomObjective.Seconds = 4
	r.tickCollapseObjective(.02)
	hp := r.Player.HP
	for n := 0; n < 20; n++ {
		r.tickCollapseObjective(.02)
	}
	if r.Player.HP != hp {
		t.Fatal("collapse damage repeated before cooldown")
	}
	for n := 0; n < 31; n++ {
		r.tickCollapseObjective(.02)
	}
	if r.Player.HP >= hp {
		t.Fatal("collapse damage did not repeat")
	}
	r.Player.X, r.Player.Y = r.RoomObjective.Zone.X, r.RoomObjective.Zone.Y
	r.Player.Jump = .5
	r.tickCollapseObjective(.02)
	if r.RoomObjective.Complete {
		t.Fatal("airborne player escaped before landing")
	}
	r.Player.Jump = 0
	r.tickCollapseObjective(.02)
	if !r.RoomObjective.Complete {
		t.Fatal("landed player could not escape")
	}
}
func TestCollapseEmptyPatrolStillRequiresExit(t *testing.T) {
	r := collapseTestRun()
	r.Enemies[0].HP = 0
	r.tick(Input{}, .02)
	if r.Status != "fighting" || r.RoomObjective.Complete {
		t.Fatal("patrol defeat bypassed escape")
	}
}

func TestCollapseCampaignPlacement(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			want := level.ID%10 == 5 && room == 0
			if (arena.Objective == "escape_collapse") != want {
				t.Fatalf("wrong collapse placement %d/%d", level.ID, room)
			}
			if want {
				count++
			}
		}
	}
	if count != 10 {
		t.Fatal("missing collapse region")
	}
}

func TestCollapseRetainsEarnedLootOnEscape(t *testing.T) {
	r := collapseTestRun()
	r.hurtEnemy(0, 1000, "hit")
	if len(r.Drops) != 1 {
		t.Fatal("earned drop missing")
	}
	r.Player.X, r.Player.Y = r.RoomObjective.Zone.X, r.RoomObjective.Zone.Y
	r.tick(Input{}, .02)
	if r.Status != "cleared" || r.Stats.Kills != 1 || len(r.Drops) != 1 || !r.Drops[0].Collected {
		t.Fatal("escape lost earned loot")
	}
}
