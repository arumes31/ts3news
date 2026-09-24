package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMissionBestsSeparateVersionsWithoutLosingProgress(t *testing.T) {
	r := NewRunAtLevel("versions", Build{HP: 100, Class: "vanguard"}, time.Unix(100, 0), nil, 1)
	finish := func(seconds, hp float64, hits int) {
		r.Stats.Seconds += seconds
		r.Player.HP = hp
		r.Stats.HitsTaken += hits
		r.finishMissionHistory("completed")
	}
	finish(10, 90, 0)
	first := r.History[1]
	r.Level.Rooms[0].Hazards[0].Period++
	r.beginMissionHistory()
	finish(20, 50, 3)
	changed := r.History[1]
	if changed.BestSeconds != 20 || changed.BestFinishHP != 50 || *changed.FewestHits != 3 || changed.Versions[first.Definition].BestSeconds != 10 {
		t.Fatal("different mission definitions competed")
	}
	if changed.Completions != 2 || changed.CompletedByClass["vanguard"] != 2 {
		t.Fatal("lost lifetime progress")
	}
	r.Level.Rooms[0].Hazards[0].Period--
	r.beginMissionHistory()
	finish(30, 40, 4)
	if r.History[1].BestSeconds != 10 || r.History[1].BestFinishHP != 90 || *r.History[1].FewestHits != 0 {
		t.Fatal("returning definition lost best")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	next := NewRunAtLevel("next", Build{HP: 100}, time.Unix(200, 0), nil, 1)
	next.InheritCampaignHistory(&saved)
	if next.History[1].Completions != 3 || next.History[1].BestSeconds != 10 {
		t.Fatal("inheritance lost matching results")
	}
	next.Level.Rooms[0].Hazards[0].Period += 2
	next.beginMissionHistory()
	if next.History[1].BestSeconds != 0 || saved.History[1].BestSeconds != 10 {
		t.Fatal("changed inheritance retained incompatible best or mutated source")
	}
	legacy := MissionHistory{BestSeconds: 1, Completions: 2}
	legacy.selectDefinition("new")
	if legacy.BestSeconds != 0 || legacy.Versions[""].BestSeconds != 1 || legacy.Completions != 2 {
		t.Fatal("legacy best competed or was lost")
	}
}
