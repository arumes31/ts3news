package rift

import (
	"encoding/json"
	"testing"
)

func guardiansTestRun() *Run {
	r := circleTestRun()
	r.Level.Rooms[0].Objective = "linked_guardians"
	r.Enemies = []Actor{
		{ID: "left", Kind: "goblin", X: 500, Y: 330, HP: 100, MaxHP: 100, Knockdown: 100},
		{ID: "right", Kind: "archer", X: 650, Y: 330, HP: 100, MaxHP: 100, Knockdown: 100},
		{ID: "patrol", Kind: "goblin", X: 1400, Y: 490, HP: 100, MaxHP: 100, Knockdown: 100},
	}
	r.beginRoomObjective()
	return r
}
func TestGuardianBondReducesDamageOnlyWhileClose(t *testing.T) {
	r := guardiansTestRun()
	if r.RoomObjective == nil {
		t.Fatal("guardians missing")
	}
	r.hurtEnemy(0, 20, "hit")
	if r.Enemies[0].HP != 90 || r.Stats.DamageDealt != 10 {
		t.Fatal("bond did not halve actual damage")
	}
	r.Enemies[1].X = 900
	r.hurtEnemy(0, 20, "hit")
	if r.Enemies[0].HP != 70 {
		t.Fatal("separated guardian remained protected")
	}
	r.Enemies[1].X = 650
	r.hurtEnemy(2, 20, "hit")
	if r.Enemies[2].HP != 80 {
		t.Fatal("patrol received bond protection")
	}
}
func TestGuardianDeathBreaksBondAndPatrolStillGatesClear(t *testing.T) {
	r := guardiansTestRun()
	if r.RoomObjective == nil {
		t.Fatal("guardians missing")
	}
	r.hurtEnemy(0, 1000, "fire")
	r.hurtEnemy(1, 20, "hit")
	if r.Enemies[1].HP != 80 {
		t.Fatal("dead guardian still protected partner")
	}
	r.hurtEnemy(1, 1000, "hit")
	r.tick(Input{}, .02)
	if !r.RoomObjective.Complete || r.Status != "fighting" || r.Stats.Kills != 2 || len(r.Drops) != 2 {
		t.Fatal("guardian rewards or patrol gate incorrect")
	}
	r.hurtEnemy(2, 1000, "hit")
	r.tick(Input{}, .02)
	if r.Status != "cleared" || r.Stats.Kills != 3 {
		t.Fatal("room did not clear normally")
	}
}
func TestGuardianTargetsAndDamagePersist(t *testing.T) {
	r := guardiansTestRun()
	if r.RoomObjective == nil {
		t.Fatal("guardians missing")
	}
	r.hurtEnemy(0, 20, "hit")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	saved.hurtEnemy(0, 20, "hit")
	if saved.Enemies[0].HP != 80 || len(saved.RoomObjective.Targets) != 2 {
		t.Fatal("saved bond or damage lost")
	}
}

func TestGuardianBondReformsAndUsesExactRange(t *testing.T) {
	r := guardiansTestRun()
	r.Enemies[1].X = 740
	r.updateGuardianObjective()
	if !r.RoomObjective.BondActive {
		t.Fatal("bond lost at exact boundary")
	}
	r.Enemies[1].X = 740.1
	r.updateGuardianObjective()
	if r.RoomObjective.BondActive {
		t.Fatal("bond remained outside range")
	}
	r.Enemies[1].X = 650
	r.updateGuardianObjective()
	if !r.RoomObjective.BondActive {
		t.Fatal("bond did not reform")
	}
}
func TestGuardianSelectionExcludesEscapersAndProps(t *testing.T) {
	r := guardiansTestRun()
	r.Enemies = append([]Actor{{ID: "treasure", Kind: "treasure", HP: 10}, {ID: "prop", Kind: "totem", HP: 10}, {ID: "boss", Kind: "boss", HP: 10}}, r.Enemies...)
	r.beginRoomObjective()
	if r.RoomObjective.Targets[0] != "left" || r.RoomObjective.Targets[1] != "right" {
		t.Fatal("invalid guardian selected")
	}
	r.Enemies = r.Enemies[:4]
	r.beginRoomObjective()
	if r.RoomObjective != nil {
		t.Fatal("incomplete pair became mandatory")
	}
}

func TestGuardianCampaignPlacement(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			want := level.ID%10 == 6 && room == 0
			if (arena.Objective == "linked_guardians") != want {
				t.Fatalf("wrong guardians placement %d/%d", level.ID, room)
			}
			if want {
				count++
			}
		}
	}
	if count != 10 {
		t.Fatal("missing guardian region")
	}
}
