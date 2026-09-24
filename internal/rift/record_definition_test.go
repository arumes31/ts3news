package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestRecordDefinitionFrozenThroughSaveAndCompletion(t *testing.T) {
	r := NewRunAtLevel("definition", Build{HP: 100}, time.Unix(100, 0), content.AbyssMobCatalog(), 4)
	original := r.MissionDefinition
	if original == "" || original != levelDefinition(r.Level) {
		t.Fatal("missing initial definition")
	}
	encoded, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err := json.Unmarshal(encoded, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.MissionDefinition != original {
		t.Fatal("definition lost in save")
	}
	saved.Level.Rooms[0].Obstacles[0].X++
	if levelDefinition(saved.Level) == original {
		t.Fatal("terrain change not detected")
	}
	saved.Stats.Seconds = 12
	saved.finishMissionHistory("completed")
	if saved.AttemptHistory[0].Definition != original {
		t.Fatal("attempt used changed content rather than start snapshot")
	}
	saved.beginMissionHistory()
	if saved.MissionDefinition == original {
		t.Fatal("new attempt did not capture changed definition")
	}
	saved.MissionDefinition = ""
	if saved.attemptSnapshot("expired", 1).Definition != "" {
		t.Fatal("legacy version inferred")
	}
}

func TestRecordDefinitionStableAndUnknown(t *testing.T) {
	a, b := Campaign()[0], Campaign()[0]
	if levelDefinition(&a) != levelDefinition(&b) {
		t.Fatal("unstable catalog identity")
	}
	b.Rooms[0].Hazards[0].Period++
	if levelDefinition(&a) == levelDefinition(&b) {
		t.Fatal("hazard change not detected")
	}
	b.Rooms[0].Hazards[0].Period = math.NaN()
	if levelDefinition(nil) != "" || levelDefinition(&b) != "" {
		t.Fatal("invalid definition identified")
	}
}

func TestLevelJSONExposesStableDefinition(t *testing.T) {
	level := Campaign()[0]
	expected := levelDefinition(&level)
	raw, err := json.Marshal(level)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Definition string `json:"definition"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Definition != expected {
		t.Fatal("wire definition does not match record identity")
	}
	var decoded Level
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if levelDefinition(&decoded) != expected {
		t.Fatal("JSON roundtrip changed content identity")
	}
}
