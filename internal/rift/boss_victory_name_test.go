package rift

import (
	"encoding/json"
	"testing"
)

func TestBossVictoryEventPreservesDefeatedIdentity(t *testing.T) {
	r := testRun()
	r.Enemies = []Actor{{ID: "boss", Name: "The Mossbound King", Kind: "boss", ArtKey: "monster:test", HP: 100, MaxHP: 100}}
	r.hurtEnemy(0, 1000, "hit")
	data, err := json.Marshal(r.Events)
	if err != nil {
		t.Fatal(err)
	}
	var events []struct {
		Kind      string `json:"kind"`
		ActorName string `json:"actor_name"`
	}
	if err := json.Unmarshal(data, &events); err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Kind == "boss_death" {
			if event.ActorName != "The Mossbound King" {
				t.Fatal("boss victory event lost defeated identity")
			}
			return
		}
	}
	t.Fatal("missing boss victory event")
}
