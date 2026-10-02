package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestBestiaryRecordsTrackOnlySpawnedCampaignMonstersAndActualDefeats(t *testing.T) {
	catalog := []content.Mob{{Name: "Record minion", Type: content.MobCommon}, {Name: "Record boss", Type: content.MobBoss}}
	r := NewRunAtLevel("records", testRun().Build, time.Unix(100, 0), catalog, 1)
	if len(r.MonsterRecords) != 1 || r.MonsterRecords["monster:Record minion"].FirstSeenMS != 100000 {
		t.Fatalf("invalid first-room records: %+v", r.MonsterRecords)
	}
	if _, ok := r.MonsterRecords["monster:Record boss"]; ok {
		t.Fatal("unseen boss was recorded")
	}
	r.hurtEnemy(0, 1e9, "hit")
	r.hurtEnemy(0, 1e9, "hit")
	if r.MonsterRecords["monster:Record minion"].Defeats != 1 {
		t.Fatal("defeat was not recorded exactly once")
	}
	saved, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err := json.Unmarshal(saved, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.MonsterRecords["monster:Record minion"].Defeats != 1 {
		t.Fatal("monster records did not survive save round-trip")
	}
	next := NewRunAtLevel("next", r.Build, time.Unix(200, 0), catalog, 1)
	next.InheritCampaignHistory(&restored)
	if next.MonsterRecords["monster:Record minion"].FirstSeenMS != 100000 || next.MonsterRecords["monster:Record minion"].Defeats != 1 {
		t.Fatal("new expedition lost monster history")
	}
	next.hurtEnemy(0, 1e9, "hit")
	if next.MonsterRecords["monster:Record minion"].Defeats != 2 || restored.MonsterRecords["monster:Record minion"].Defeats != 1 {
		t.Fatal("record inheritance shared mutable state")
	}
	next.Room = 2
	next.LastMS = 300000
	next.spawnRoom()
	if next.MonsterRecords["monster:Record boss"].FirstSeenMS != 300000 {
		t.Fatal("boss appearance was not recorded at spawn")
	}
}

func TestPracticeAndLegacyUnknownsDoNotInventBestiaryHistory(t *testing.T) {
	practice, _ := NewPracticeRun("practice", testRun().Build, "combo", time.Unix(100, 0))
	practice.hurtEnemy(0, 1e9, "hit")
	if len(practice.MonsterRecords) != 0 {
		t.Fatal("practice created campaign monster records")
	}
	old := NewRunAtLevel("old", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
	old.MonsterRecords = nil
	old.Step(Input{}, time.Unix(200, 0))
	for _, record := range old.MonsterRecords {
		if record.FirstSeenMS != 200000 || record.Defeats != 0 {
			t.Fatal("legacy history was backfilled from unrecorded combat")
		}
	}
	if len(old.MonsterRecords) == 0 {
		t.Fatal("legacy current encounters were never observed")
	}
}
