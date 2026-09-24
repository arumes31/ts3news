package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestBossClearRecordWaitsForWholeRoomAndKeepsFastest(t *testing.T) {
	catalog := []content.Mob{{Name: "Timed boss", Type: content.MobBoss}, {Name: "Minion", Type: content.MobCommon}}
	var previous *Run
	for _, seconds := range []float64{12, 18, 8} {
		r := NewRunAtLevel("timed", testRun().Build, time.Unix(100, 0), catalog, 1)
		r.InheritCampaignHistory(previous)
		r.Room = 2
		r.spawnRoom()
		r.Stats.Seconds = seconds
		r.hurtEnemy(0, 1e9, "hit")
		if previous == nil && r.MonsterRecords["monster:Timed boss"].FastestClearSeconds != nil {
			t.Fatal("boss death recorded before room clear")
		}
		for i := 1; i < len(r.Enemies); i++ {
			r.hurtEnemy(i, 1e9, "hit")
		}
		r.tick(Input{}, .02)
		record := r.MonsterRecords["monster:Timed boss"]
		want := seconds + .02
		if seconds == 18 {
			want = 12.02
		}
		if record.FastestClearSeconds == nil || *record.FastestClearSeconds != want {
			t.Fatalf("fastest=%v want %v", record.FastestClearSeconds, want)
		}
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved Run
		if err = json.Unmarshal(data, &saved); err != nil {
			t.Fatal(err)
		}
		previous = &saved
	}
}

func TestBossClearRecordRejectsPracticeAndUnknownTiming(t *testing.T) {
	for _, practice := range []bool{false, true} {
		r := testRun()
		r.RoomStartSeconds = nil
		r.Level = &Level{ID: 1}
		r.Enemies = []Actor{{Kind: "boss", Name: "Boss", ArtKey: "monster:Boss"}}
		if practice {
			r.Practice = &PracticeState{Mode: "boss"}
			start := 0.0
			r.RoomStartSeconds = &start
		}
		r.Status = "cleared"
		r.recordBossClear()
		if len(r.MonsterRecords) != 0 {
			t.Fatal("unmeasured or practice clear created record")
		}
	}
}

func TestBossClearRecordsRemainIndependentByCanonicalIdentity(t *testing.T) {
	r := testRun()
	r.Level = &Level{ID: 1}
	r.Status = "cleared"
	start := 0.0
	r.RoomStartSeconds = &start
	clear := func(name string, seconds float64) {
		r.Enemies = []Actor{{Kind: "boss", Name: name, ArtKey: "monster:" + name}}
		r.Stats.Seconds = seconds
		r.recordBossClear()
	}
	clear("First boss", 12)
	clear("Second boss", 6)
	clear("First boss", 20)
	if *r.MonsterRecords["monster:First boss"].FastestClearSeconds != 12 || *r.MonsterRecords["monster:Second boss"].FastestClearSeconds != 6 {
		t.Fatal("boss identities shared a record")
	}
	clear("Second boss", 4)
	if *r.MonsterRecords["monster:First boss"].FastestClearSeconds != 12 || *r.MonsterRecords["monster:Second boss"].FastestClearSeconds != 4 {
		t.Fatal("improving one boss changed another")
	}
	r.Enemies = []Actor{{Kind: "boss", Name: "Untracked", ArtKey: "invalid"}}
	r.recordBossClear()
	if len(r.MonsterRecords) != 2 {
		t.Fatal("noncanonical identity gained a record")
	}
}
