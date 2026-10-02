package rift

import (
	"reflect"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestBossRetryRestoresFrozenRoomAndPreservesBankedProgress(t *testing.T) {
	r := NewRunAtLevel("retry", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 10)
	r.Room = 2
	r.spawnRoom()
	plan := append([]Actor(nil), r.Enemies...)
	r.BankedGold = 50
	r.BankedItems = []string{"Banked sword"}
	r.BankedAtMS = 99000
	r.Stats.Seconds = 45
	r.Stats.HitsTaken = 4
	r.Player.HP = 0
	r.tick(Input{}, .02)
	if r.Status != "defeated" {
		t.Fatal("fixture did not defeat player")
	}
	missionStart := r.MissionStartSeconds
	missionHits := *r.MissionStartHits
	r.Combo = 2
	r.ComboTime = 1
	r.Resource = 10
	r.Barrier = 40
	r.Marked = "target"
	r.SkillTimers["fire"] = 3
	r.Player.Jump = .5
	r.Player.Knockdown = 1
	r.jumpAir = 3
	r.jumpDist = 5
	r.heavyRecovery = 2
	r.Revision = 9
	r.Epoch = "epoch"
	r.StartKey = "start"
	r.Gold = 99
	r.Drops = []Drop{{Gold: 99}}
	if err := r.RetryBossEncounter(time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	if r.Room != 2 || !reflect.DeepEqual(r.Enemies, plan) || r.Status != "fighting" || !r.Paused {
		t.Fatal("retry did not restore paused frozen room")
	}
	if r.Player.HP != r.Player.MaxHP || r.Player.Mana != 100 || r.Player.Jump != 0 || r.Player.Knockdown != 0 || r.Combo != 0 || r.Resource != 0 || r.Barrier != 0 || r.Marked != "" || len(r.SkillTimers) != 0 || r.jumpAir != 0 || r.jumpDist != 0 || r.heavyRecovery != 0 {
		t.Fatal("retry retained combat state")
	}
	if r.BankedGold != 50 || len(r.BankedItems) != 1 || r.BankedAtMS != 99000 || r.Gold != 0 || len(r.Drops) != 0 || r.Revision != 9 || r.Epoch != "epoch" || r.StartKey != "start" {
		t.Fatal("retry changed banked rewards or request identity")
	}
	if r.Stats.Seconds < 45 || r.Stats.HitsTaken != 4 || r.MissionStartSeconds != missionStart || *r.MissionStartHits != missionHits || !r.HistoryActive || r.History[10].Attempts != 2 {
		t.Fatal("retry discarded failed-attempt accounting")
	}
	if err := r.RetryBossEncounter(time.Unix(201, 0)); err == nil {
		t.Fatal("active encounter retried twice")
	}
}

func TestBossRetryRejectsOtherRunStates(t *testing.T) {
	for _, status := range []string{"fighting", "cleared", "complete", "banked", "expired"} {
		r := NewRunAtLevel("reject", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
		r.Room = 2
		r.Status = status
		if r.RetryBossEncounter(time.Now()) == nil {
			t.Fatalf("accepted %s", status)
		}
	}
	r := testRun()
	r.Status = "defeated"
	if r.RetryBossEncounter(time.Now()) == nil {
		t.Fatal("legacy noncampaign accepted")
	}
	r.Level = &Level{ID: 1}
	r.EncounterPlan = nil
	if r.RetryBossEncounter(time.Now()) == nil {
		t.Fatal("missing frozen plan accepted")
	}
}
