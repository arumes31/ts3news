package rift

import (
	"encoding/json"
	"testing"
)

func TestMarkEndRecordsCauseAndResetsOnNewIntent(t *testing.T) {
	for _, reason := range []string{"defeated", "escaped"} {
		t.Run(reason, func(t *testing.T) {
			r := testRun()
			r.Enemies = []Actor{{ID: "target", Name: "Marked scout", HP: 10, MaxHP: 10}}
			r.Marked = "target"
			if reason == "defeated" {
				r.hurtEnemyPiercing(0, 20, "hit", 0)
			} else {
				r.escapeEnemy(0)
			}
			if r.Marked != "" || r.LastMarkEnd == nil || r.LastMarkEnd.Reason != reason || r.LastMarkEnd.TargetName != "Marked scout" {
				t.Fatal("mark end lost cause")
			}
			data, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var saved Run
			if err = json.Unmarshal(data, &saved); err != nil {
				t.Fatal(err)
			}
			if saved.LastMarkEnd == nil || *saved.LastMarkEnd != *r.LastMarkEnd {
				t.Fatal("receipt lost on save")
			}
			saved.Enemies = []Actor{{ID: "other", HP: 100, MaxHP: 100}}
			saved.skillHit(0, 1, Skill{Role: "builder"}, 0, "")
			if saved.LastMarkEnd != nil || saved.Marked != "other" {
				t.Fatal("new mark retained stale explanation")
			}
			r.classCast(Skill{Role: "finisher"})
			if r.LastMarkEnd != nil {
				t.Fatal("finisher retained stale explanation")
			}
		})
	}
	r := testRun()
	r.Marked = "other"
	r.Enemies = []Actor{{ID: "target", HP: 10, MaxHP: 10}}
	r.hurtEnemyPiercing(0, 20, "hit", 0)
	if r.LastMarkEnd != nil || r.Marked != "other" {
		t.Fatal("unmarked defeat changed mark")
	}
}

func TestMarkExplanationClearsAtRoomEntryAndIgnoresEmptyIDs(t *testing.T) {
	r := testRun()
	r.LastMarkEnd = &MarkEnd{TargetName: "Old target", Reason: "defeated"}
	r.Status = "cleared"
	r.NextRoom()
	if r.LastMarkEnd != nil {
		t.Fatal("previous-room explanation leaked")
	}
	r.Marked = ""
	r.endTargetMark(Actor{}, "defeated")
	if r.LastMarkEnd != nil {
		t.Fatal("empty mark created explanation")
	}
}
