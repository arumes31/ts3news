package rift

import (
	"encoding/json"
	"testing"
	"time"

	"ts3news/internal/content"
)

func TestEncounterSummaryRoomClearAndDeltas(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	now := time.Unix(1000, 0)
	run := NewRunAtLevel("test-encounter", Build{HP: 200}, now, catalog, 1)

	if run.LastEncounter != nil {
		t.Fatalf("expected LastEncounter to be nil at start, got %+v", run.LastEncounter)
	}
	if run.RoomBaseline == nil {
		t.Fatal("expected RoomBaseline to be initialized at start")
	}

	// Simulate room 0 combat actions
	run.Stats.DamageDealt += 150
	run.Stats.DamageTaken += 30
	run.Stats.HitsTaken += 2
	run.Stats.GuardBlocked += 40
	run.Stats.BarrierBlocked += 10
	run.Stats.Kills += 2
	run.Gold += 25
	run.Stats.Seconds = 14.5

	// Kill remaining enemies
	for i := range run.Enemies {
		run.Enemies[i].HP = 0
	}

	// Step combat to trigger clear
	run.Step(Input{}, now.Add(time.Second))

	if run.Status != "cleared" {
		t.Fatalf("expected status cleared, got %s", run.Status)
	}
	if run.LastEncounter == nil {
		t.Fatal("expected LastEncounter to be populated after room clear")
	}

	enc := run.LastEncounter
	if enc.Outcome != "cleared" {
		t.Errorf("expected outcome cleared, got %s", enc.Outcome)
	}
	if enc.Room != 0 {
		t.Errorf("expected room 0, got %d", enc.Room)
	}
	if enc.RoomName != run.Level.Rooms[0].Name {
		t.Errorf("expected room name %s, got %s", run.Level.Rooms[0].Name, enc.RoomName)
	}
	if enc.DamageDealt != 150 {
		t.Errorf("expected damage dealt 150, got %f", enc.DamageDealt)
	}
	if enc.DamageTaken != 30 {
		t.Errorf("expected damage taken 30, got %f", enc.DamageTaken)
	}
	if enc.HitsTaken != 2 {
		t.Errorf("expected hits taken 2, got %d", enc.HitsTaken)
	}
	if enc.GuardBlocked != 40 {
		t.Errorf("expected guard blocked 40, got %f", enc.GuardBlocked)
	}
	if enc.BarrierBlocked != 10 {
		t.Errorf("expected barrier blocked 10, got %f", enc.BarrierBlocked)
	}
	if enc.GoldGained != 25 {
		t.Errorf("expected gold gained 25, got %d", enc.GoldGained)
	}

	// Advance to room 1
	if !run.NextRoom() {
		t.Fatal("NextRoom failed")
	}
	if run.Room != 1 {
		t.Fatalf("expected room 1, got %d", run.Room)
	}

	// LastEncounter should still preserve room 0 summary while fighting room 1
	if run.LastEncounter.Room != 0 {
		t.Errorf("expected LastEncounter to preserve room 0 during room 1, got room %d", run.LastEncounter.Room)
	}

	// Simulate room 1 combat actions
	run.Stats.DamageDealt += 200 // total now 350, delta is 200
	run.Stats.DamageTaken += 50  // total now 80, delta is 50
	run.Stats.HitsTaken += 3    // total now 5, delta is 3
	run.Stats.GuardBlocked += 60 // total now 100, delta is 60
	run.Stats.Seconds = 30.0    // delta is 30 - 14.5 = 15.5
	run.Gold += 35              // total now 60, delta is 35

	for i := range run.Enemies {
		run.Enemies[i].HP = 0
	}
	run.Step(Input{}, now.Add(2*time.Second))

	if run.Status != "cleared" {
		t.Fatalf("expected room 1 status cleared, got %s", run.Status)
	}
	enc1 := run.LastEncounter
	if enc1.Room != 1 {
		t.Errorf("expected LastEncounter room 1, got %d", enc1.Room)
	}
	if enc1.RoomName != run.Level.Rooms[1].Name {
		t.Errorf("expected room name %s, got %s", run.Level.Rooms[1].Name, enc1.RoomName)
	}
	if enc1.DamageDealt != 200 {
		t.Errorf("expected room 1 delta damage dealt 200, got %f", enc1.DamageDealt)
	}
	if enc1.DamageTaken != 50 {
		t.Errorf("expected room 1 delta damage taken 50, got %f", enc1.DamageTaken)
	}
	if enc1.HitsTaken != 3 {
		t.Errorf("expected room 1 delta hits taken 3, got %d", enc1.HitsTaken)
	}
	if enc1.GuardBlocked != 60 {
		t.Errorf("expected room 1 delta guard blocked 60, got %f", enc1.GuardBlocked)
	}
	if enc1.GoldGained != 35 {
		t.Errorf("expected room 1 delta gold gained 35, got %d", enc1.GoldGained)
	}
}

func TestEncounterSummaryDefeat(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	now := time.Unix(1000, 0)
	run := NewRunAtLevel("test-defeat-encounter", Build{HP: 200}, now, catalog, 1)

	run.Stats.DamageDealt = 80
	run.Stats.DamageTaken = 200
	run.Stats.HitsTaken = 4
	run.Gold = 50
	run.Player.HP = 0

	run.Step(Input{}, now.Add(time.Second))

	if run.Status != "defeated" {
		t.Fatalf("expected status defeated, got %s", run.Status)
	}
	if run.LastEncounter == nil {
		t.Fatal("expected LastEncounter to be populated on defeat")
	}

	enc := run.LastEncounter
	if enc.Outcome != "defeated" {
		t.Errorf("expected outcome defeated, got %s", enc.Outcome)
	}
	if enc.PlayerHP != 0 {
		t.Errorf("expected ending HP 0, got %f", enc.PlayerHP)
	}
	if enc.DamageDealt != 80 {
		t.Errorf("expected damage dealt 80, got %f", enc.DamageDealt)
	}
	if enc.DamageTaken != 200 {
		t.Errorf("expected damage taken 200, got %f", enc.DamageTaken)
	}
	if enc.GoldGained != 50 {
		t.Errorf("expected lost unbanked gold 50 recorded, got %d", enc.GoldGained)
	}
}

func TestEncounterSummaryJSONSerialization(t *testing.T) {
	enc := &EncounterSummary{
		Mission:        1,
		MissionName:    "Mossbound Ruins",
		Room:           0,
		RoomName:       "Approach",
		Outcome:        "cleared",
		Seconds:        12.5,
		PlayerHP:       180.0,
		PlayerMaxHP:    200.0,
		Enemies:        3,
		BossEncounter:  false,
		DamageDealt:    250.0,
		DamageTaken:    20.0,
		HitsTaken:      1,
		GuardBlocked:   35.0,
		BarrierBlocked: 0,
		Healing:        0,
		GoldGained:     45,
		LootItems:      1,
	}

	data, err := json.Marshal(enc)
	if err != nil {
		t.Fatalf("failed to marshal EncounterSummary: %v", err)
	}

	var restored EncounterSummary
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal EncounterSummary: %v", err)
	}

	if restored.Mission != enc.Mission || restored.Outcome != enc.Outcome || restored.Seconds != enc.Seconds {
		t.Errorf("restored summary mismatch: %+v vs %+v", restored, enc)
	}
}
