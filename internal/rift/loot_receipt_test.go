package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestLootProvenanceSurvivesSeamlessMissionAndSave(t *testing.T) {
	r := NewRunAtLevel("origin", Build{HP: 200}, time.Now(), content.AbyssMobCatalog(), 10)
	drop := Drop{Mission: 10, Tier: 3, Gear: &content.Gear{Name: "Blade", Rarity: content.RarityRare, FoundBoss: "Original guardian", FoundAt: "2026-09-24T00:00:00Z"}}
	record := drop.LootReceipt()
	r.BankedLoot = []BankedLoot{record}
	r.Room = 2
	r.Status = "cleared"
	r.FinishCheckpoint("advance", content.AbyssMobCatalog())
	if r.Level.ID != 11 {
		t.Fatal("did not advance mission")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.BankedLoot) != 1 || saved.BankedLoot[0] != record || record.Mission != 10 || record.Tier != 3 || record.Origin != "Original guardian" || record.FoundAt != drop.Gear.FoundAt {
		t.Fatal("loot origin changed across mission/save")
	}
}
