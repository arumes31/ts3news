package rift

import (
	"encoding/json"
	"testing"
)

func TestEncounterSummarySeparatesEscapedTreasureFromDefeats(t *testing.T) {
	r := testRun()
	r.Enemies = []Actor{{ID: "runner", Kind: "treasure", X: 55, Y: 410, HP: 100, MaxHP: 100}, {ID: "target", Kind: "goblin", X: 400, Y: 410, HP: 10, MaxHP: 10}}
	r.EncounterPlan[r.Room] = append([]Actor{}, r.Enemies...)
	r.enemyTick(0, .02)
	r.hurtEnemy(1, 100, "hit")
	r.tick(Input{}, .02)
	if r.Status != "cleared" {
		t.Fatal("fixture did not clear encounter")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(saved.LastEncounter)
	if err != nil {
		t.Fatal(err)
	}
	var summary struct {
		Enemies int `json:"enemies"`
		Escaped int `json:"treasure_escaped"`
	}
	if err := json.Unmarshal(encoded, &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Enemies != 1 || summary.Escaped != 1 {
		t.Fatalf("escape was not distinguished from defeat: %+v", summary)
	}
	if saved.Stats.Kills != 1 || saved.Stats.TreasureGoblins != 0 || len(saved.Drops) != 1 {
		t.Fatal("escape granted defeat credit or loot")
	}
}
