package rift

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestEncounterSummaryPreservesUniqueCanonicalMonsters(t *testing.T) {
	for _, outcome := range []string{"cleared", "defeated"} {
		t.Run(outcome, func(t *testing.T) {
			r := testRun()
			r.Enemies = []Actor{
				{Name: "Rat", ArtKey: "monster:Rat", HP: 0},
				{Name: "Rat", ArtKey: "monster:Rat", HP: 20},
				{Name: "King", ArtKey: "monster:King", Kind: "boss", HP: 10},
				{Name: "Goblin", ArtKey: "monster:Goblin", Kind: "treasure", Pose: "escape"},
				{Name: "Crate", ArtKey: "prop:crate"},
				{Name: "Unknown", ArtKey: "monster:Rat"},
			}
			r.RecordEncounterSummary(outcome)
			want := []string{"monster:Rat", "monster:King", "monster:Goblin"}
			if !reflect.DeepEqual(r.LastEncounter.MonsterKeys, want) {
				t.Fatalf("encounter identities=%v want %v", r.LastEncounter.MonsterKeys, want)
			}
			r.Enemies[0].ArtKey = "monster:Other"
			r.spawnRoom()
			data, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var saved Run
			if err := json.Unmarshal(data, &saved); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(saved.LastEncounter.MonsterKeys, want) {
				t.Fatal("next room or save changed encounter identities")
			}
			next := NewRunAtLevel("next", r.Build, time.Unix(1000, 0), content.AbyssMobCatalog(), 2)
			next.InheritCampaignHistory(&saved)
			if !reflect.DeepEqual(next.LastEncounter.MonsterKeys, want) {
				t.Fatal("campaign continuation lost encounter identities")
			}
		})
	}
}
