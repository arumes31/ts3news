package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRegionalRecordsRequireWholeOrderedRunAndSurviveSave(t *testing.T) {
	r := NewRunAtLevel("regions", Build{HP: 200}, time.Unix(100, 0), nil, 1)
	finish := func(id int, seconds float64, outcome string) {
		r.Level.ID = id
		r.beginMissionHistory()
		r.Stats.Seconds += seconds
		r.finishMissionHistory(outcome)
	}
	for id := 1; id <= 9; id++ {
		finish(id, 10, "completed")
	}
	if len(r.RegionRecords) != 0 {
		t.Fatal("partial region recorded")
	}
	raw, _ := json.Marshal(r)
	var saved Run
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	r = &saved
	finish(10, 10, "completed")
	if r.RegionRecords[0].BestSeconds != 100 {
		t.Fatal("wrong full region time")
	}
	r.finishMissionHistory("completed")
	if r.RegionRecords[0].BestSeconds != 100 {
		t.Fatal("replay changed time")
	}
	for id := 1; id <= 10; id++ {
		finish(id, 12, "completed")
	}
	if r.RegionRecords[0].BestSeconds != 100 {
		t.Fatal("slower time replaced best")
	}
	for id := 1; id <= 10; id++ {
		finish(id, 5, "completed")
	}
	if r.RegionRecords[0].BestSeconds != 50 {
		t.Fatal("faster time not recorded")
	}
	next := NewRunAtLevel("next", Build{HP: 200}, time.Unix(200, 0), nil, 1)
	next.InheritCampaignHistory(r)
	next.RegionRecords[0] = RegionRecord{BestSeconds: 40}
	if r.RegionRecords[0].BestSeconds != 50 || next.RegionAttempt != nil {
		t.Fatal("inheritance aliases or carries attempt")
	}
	for _, outcome := range []string{"defeated", "exited"} {
		finish(11, 1, "completed")
		finish(12, 1, outcome)
		for id := 12; id <= 20; id++ {
			finish(id, 1, "completed")
		}
		if _, ok := r.RegionRecords[1]; ok {
			t.Fatal("broken attempt recorded", outcome)
		}
	}
	finish(21, 1, "completed")
	finish(23, 1, "completed")
	if r.RegionAttempt != nil {
		t.Fatal("skipped mission retained attempt")
	}
}

func TestRegionalRecordsSeparateDefinitionsAndRetainLegacy(t *testing.T) {
	r := NewRunAtLevel("versions", Build{HP: 200}, time.Unix(100, 0), nil, 1)
	r.RegionRecords = map[int]RegionRecord{0: {BestSeconds: 1, AtMS: 5}}
	finish := func(seconds float64) {
		for id := 1; id <= 10; id++ {
			r.Level.ID = id
			r.beginMissionHistory()
			r.Stats.Seconds += seconds
			r.finishMissionHistory("completed")
		}
	}
	finish(10)
	first := r.RegionRecords[0]
	if first.BestSeconds != 100 || first.Definition == "" || r.RegionVersions[0][""].BestSeconds != 1 {
		t.Fatal("legacy record competed or was lost")
	}
	r.Level.Rooms[0].Hazards[0].Period++
	finish(20)
	changed := r.RegionRecords[0]
	if changed.BestSeconds != 200 || changed.Definition == first.Definition || r.RegionVersions[0][first.Definition] != first {
		t.Fatal("different definitions competed or lost record")
	}
	r.Level.Rooms[0].Hazards[0].Period--
	finish(30)
	if r.RegionRecords[0] != first {
		t.Fatal("returning definition lost earlier best")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	next := NewRunAtLevel("inherit", Build{HP: 200}, time.Unix(200, 0), nil, 1)
	next.InheritCampaignHistory(&saved)
	delete(next.RegionVersions[0], first.Definition)
	if saved.RegionVersions[0][first.Definition] != first {
		t.Fatal("inherited versions alias")
	}
	r.Level.ID = 1
	r.beginMissionHistory()
	r.MissionDefinition = ""
	r.recordRegionTime(10)
	if r.RegionAttempt != nil {
		t.Fatal("unknown definition started regional comparison")
	}
}
