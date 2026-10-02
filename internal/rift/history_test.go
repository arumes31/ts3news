package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestMissionHistoryLifecycleAndCarry(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	r := NewRunAtLevel("first", Build{HP: 100}, time.Unix(100, 0), catalog, 4)
	if h := r.History[4]; h.Attempts != 1 || h.LastStartedMS != 100000 || h.LastOutcome != "active" {
		t.Fatalf("entry: %+v", h)
	}
	r.Stats.Seconds = 12
	r.Status = "cleared"
	r.FinishCheckpoint("next", catalog)
	if r.History[4].Attempts != 1 || !r.HistoryActive {
		t.Fatal("tier transition counted a new mission")
	}
	r.Stats.Seconds = 35
	r.Clock = 1000
	r.Room = 2
	r.Status = "cleared"
	r.LastMS = 200000
	r.FinishCheckpoint("advance", catalog)
	if h := r.History[4]; h.Completions != 1 || h.BestSeconds != 35 || h.LastOutcome != "completed" {
		t.Fatalf("completion: %+v", h)
	}
	if h := r.History[5]; h.Attempts != 1 || h.LastStartedMS != 200000 || r.MissionStartSeconds != 35 {
		t.Fatalf("advance: %+v", h)
	}
	r.Stats.Seconds = 42
	r.Status = "cleared"
	r.FinishCheckpoint("exit", catalog)
	if h := r.History[5]; h.Completions != 0 || h.BestSeconds != 0 || h.LastOutcome != "exited" {
		t.Fatalf("exit: %+v", h)
	}
	r.FinishCheckpoint("exit", catalog)
	encoded, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(encoded, &saved); err != nil {
		t.Fatal(err)
	}
	next := NewRunAtLevel("second", Build{HP: 100}, time.Unix(300, 0), catalog, 4)
	next.InheritCampaignHistory(&saved)
	if h := next.History[4]; h.Attempts != 2 || h.Completions != 1 || h.BestSeconds != 35 || h.LastOutcome != "active" {
		t.Fatalf("carry: %+v", h)
	}
	next.Stats.Seconds = 20
	next.Room = 2
	next.Status = "cleared"
	next.FinishCheckpoint("bank", catalog)
	if h := next.History[4]; h.Completions != 2 || h.BestSeconds != 20 {
		t.Fatalf("best: %+v", h)
	}
	if saved.History[4].Attempts != 1 {
		t.Fatal("new run mutated previous history")
	}
	if len(next.CompletedLevels) != 1 || next.CompletedLevels[0] != 4 {
		t.Fatal("completion marks lost")
	}
}

func TestMissionHistoryFinishHealthPrecedesRecoveryAndKeepsBest(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	r := NewRunAtLevel("first", Build{HP: 240}, time.Unix(100, 0), catalog, 4)
	r.Status = "cleared"
	r.Room = 2
	r.Player.HP = 120
	r.FinishCheckpoint("advance", catalog)
	if r.History[4].BestFinishHP != 120 || r.History[4].BestFinishMaxHP != 240 || r.Player.HP != 180 {
		t.Fatalf("record included recovery: %+v", r.History[4])
	}
	for i, hp := range []float64{100, 160} {
		next := NewRunAtLevel("next", Build{HP: 240}, time.Unix(int64(110+i), 0), catalog, 4)
		next.InheritCampaignHistory(r)
		next.Status = "cleared"
		next.Room = 2
		next.Player.HP = hp
		next.FinishCheckpoint("bank", catalog)
		want := 120.0
		if hp > 120 {
			want = hp
		}
		if next.History[4].BestFinishHP != want {
			t.Fatalf("best health lost: %+v", next.History[4])
		}
		r = next
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.History[4].BestFinishHP != 160 {
		t.Fatal("finish health lost after save")
	}
	next := NewRunAtLevel("early", Build{HP: 240}, time.Unix(120, 0), catalog, 4)
	next.InheritCampaignHistory(&restored)
	next.Status = "cleared"
	next.FinishCheckpoint("exit", catalog)
	if next.History[4].BestFinishHP != 160 {
		t.Fatal("early exit established a health record")
	}
}

func TestMissionHistorySubclassClearsUseFrozenBuildAndDoNotAlias(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	first := NewRunAtLevel("first", Build{HP: 240, Class: "vanguard"}, time.Unix(100, 0), catalog, 4)
	first.Status = "cleared"
	first.Room = 2
	first.FinishCheckpoint("bank", catalog)
	next := NewRunAtLevel("next", Build{HP: 240, Class: "oracle"}, time.Unix(110, 0), catalog, 4)
	next.InheritCampaignHistory(first)
	next.Status = "cleared"
	next.Room = 2
	next.FinishCheckpoint("bank", catalog)
	next.FinishCheckpoint("bank", catalog)
	if classes := next.History[4].CompletedByClass; classes["vanguard"] != 1 || classes["oracle"] != 1 {
		t.Fatalf("class clears incorrect: %+v", classes)
	}
	if len(first.History[4].CompletedByClass) != 1 {
		t.Fatal("later completion mutated previous snapshot")
	}
	data, err := json.Marshal(next)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.History[4].CompletedByClass["oracle"] != 1 {
		t.Fatal("class records lost on save")
	}
	early := NewRunAtLevel("early", Build{HP: 240, Class: "marksman"}, time.Unix(120, 0), catalog, 4)
	early.InheritCampaignHistory(&restored)
	early.Status = "cleared"
	early.FinishCheckpoint("exit", catalog)
	if len(early.History[4].CompletedByClass) != 2 {
		t.Fatal("early exit counted as subclass clear")
	}
}

func TestMissionHistoryFewestHitsIncludesZeroAndExcludesPartialLegacy(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	var previous *Run
	for i, hits := range []int{2, 4, 0} {
		r := NewRunAtLevel("attempt", Build{HP: 240}, time.Unix(int64(100+i), 0), catalog, 4)
		r.InheritCampaignHistory(previous)
		r.Stats.HitsTaken = hits
		r.Status = "cleared"
		r.Room = 2
		r.FinishCheckpoint("bank", catalog)
		want := 2
		if hits == 0 {
			want = 0
		}
		if r.History[4].FewestHits == nil || *r.History[4].FewestHits != want {
			t.Fatalf("wrong best hit count: %+v", r.History[4])
		}
		previous = r
	}
	data, err := json.Marshal(previous)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.History[4].FewestHits == nil || *restored.History[4].FewestHits != 0 {
		t.Fatal("zero record lost after saving")
	}
	advance := NewRunAtLevel("advance", Build{HP: 240}, time.Unix(109, 0), catalog, 6)
	advance.Stats.HitsTaken = 5
	advance.Status = "cleared"
	advance.Room = 2
	advance.FinishCheckpoint("advance", catalog)
	advance.Stats.HitsTaken = 6
	advance.Status = "cleared"
	advance.Room = 2
	advance.FinishCheckpoint("bank", catalog)
	if advance.History[7].FewestHits == nil || *advance.History[7].FewestHits != 1 {
		t.Fatal("hits carried across mission boundary")
	}
	legacy := NewRunAtLevel("legacy", Build{HP: 240}, time.Unix(110, 0), catalog, 5)
	legacy.MissionStartHits = nil
	legacy.Status = "cleared"
	legacy.Room = 2
	legacy.FinishCheckpoint("bank", catalog)
	if legacy.History[5].FewestHits != nil {
		t.Fatal("partial legacy attempt invented a flawless record")
	}
}

func TestFlawlessRoomRecordsUseDamageAndPersistAcrossTiers(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	r := NewRunAtLevel("rooms", Build{HP: 240}, time.Unix(100, 0), catalog, 1)
	for room := 0; room < 3; room++ {
		r.Enemies = nil
		r.Level.Rooms[room].Hazards = nil
		if room == 1 {
			r.hurtPlayer(10, 0, 0)
		}
		if room == 2 {
			r.Barrier = 100
			r.hurtPlayer(10, 0, 0)
		}
		r.tick(Input{}, 1.0/30)
		r.tick(Input{}, 1.0/30)
		if room < 2 {
			if !r.NextRoom() {
				t.Fatal("room did not advance")
			}
		}
	}
	if tiers := r.History[1].FlawlessTiers; len(tiers) != 2 || tiers[0] != 1 || tiers[1] != 3 {
		t.Fatalf("incorrect flawless tiers: %v", tiers)
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	next := NewRunAtLevel("next", Build{HP: 240}, time.Unix(110, 0), catalog, 1)
	next.InheritCampaignHistory(&restored)
	if len(next.History[1].FlawlessTiers) != 2 {
		t.Fatal("records lost across expeditions")
	}
	legacy := NewRunAtLevel("legacy", Build{HP: 240}, time.Unix(120, 0), catalog, 2)
	legacy.RoomStartHits = nil
	legacy.Enemies = nil
	legacy.Level.Rooms[0].Hazards = nil
	legacy.tick(Input{}, 1.0/30)
	if len(legacy.History[2].FlawlessTiers) != 0 {
		t.Fatal("partial legacy room counted as flawless")
	}
}

func TestMissionClearStreakCarriesCompletedRunsAndBreaksOnIncompleteAttempts(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	r := NewRunAtLevel("streak", Build{HP: 240}, time.Unix(100, 0), catalog, 4)
	r.Status = "cleared"
	r.Room = 2
	r.FinishCheckpoint("advance", catalog)
	r.Status = "cleared"
	r.Room = 2
	r.FinishCheckpoint("bank", catalog)
	r.FinishCheckpoint("bank", catalog)
	if r.ClearStreak != 2 || r.BestClearStreak != 2 {
		t.Fatalf("streak not counted exactly once: %+v", r)
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	next := NewRunAtLevel("next", Build{HP: 240}, time.Unix(110, 0), catalog, 1)
	next.InheritCampaignHistory(&saved)
	if next.ClearStreak != 2 || next.BestClearStreak != 2 {
		t.Fatal("completed-run streak lost")
	}
	next.Status = "cleared"
	next.FinishCheckpoint("exit", catalog)
	if next.ClearStreak != 0 || next.BestClearStreak != 2 {
		t.Fatal("early exit did not break current streak")
	}
	next = NewRunAtLevel("abandoned", Build{HP: 240}, time.Unix(120, 0), catalog, 1)
	next.InheritCampaignHistory(&saved)
	resumed := NewRunAtLevel("replacement", Build{HP: 240}, time.Unix(130, 0), catalog, 2)
	resumed.InheritCampaignHistory(next)
	if resumed.ClearStreak != 0 || resumed.BestClearStreak != 2 {
		t.Fatal("unfinished attempt extended streak")
	}
	next.Player.HP = 0
	next.tick(Input{}, 1.0/30)
	if next.ClearStreak != 0 || next.BestClearStreak != 2 {
		t.Fatal("defeat failed to retain best and reset current")
	}
}

func TestMissionHistoryDefeatPauseLegacyAndExpired(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	now := time.Unix(100, 0)
	r := NewRunAtLevel("defeat", Build{HP: 100}, now, catalog, 1)
	r.Paused = true
	r.Step(Input{}, now.Add(time.Hour))
	if r.Clock != 0 || r.History[1].LastOutcome != "active" {
		t.Fatal("pause changed timing or outcome")
	}
	r.Paused = false
	r.Player.HP = 0
	r.Step(Input{}, now.Add(time.Hour+time.Second))
	if h := r.History[1]; h.LastOutcome != "defeated" || h.Completions != 0 || h.BestSeconds != 0 {
		t.Fatalf("defeat: %+v", h)
	}
	legacy := NewRunAtLevel("legacy", Build{HP: 100}, now, catalog, 2)
	legacy.History = nil
	legacy.HistoryActive = false
	legacy.Room = 2
	legacy.Clock = 90
	legacy.Status = "cleared"
	legacy.FinishCheckpoint("exit", catalog)
	if len(legacy.History) != 0 {
		t.Fatal("invented history for legacy run")
	}
	active := NewRunAtLevel("expired", Build{HP: 100}, now, catalog, 3)
	fresh := NewRunAtLevel("fresh", Build{HP: 100}, now, catalog, 4)
	fresh.InheritCampaignHistory(active)
	if fresh.History[3].LastOutcome != "expired" || active.History[3].LastOutcome != "active" {
		t.Fatal("expired attempt carry failed")
	}
}

func TestCareerTotalsCarryEachExpeditionOnce(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	first := NewRunAtLevel("first", Build{HP: 100}, time.Unix(100, 0), catalog, 1)
	first.Stats.Kills = 12
	first.Stats.Bosses = 2
	first.Stats.TreasureGoblins = 1
	first.BankedGold = 123
	first.BankedItems = []string{"Sword", "Sword"}
	first.Gold = 900 // Unbanked rewards must never enter career totals.
	first.Status = "defeated"
	second := NewRunAtLevel("second", Build{HP: 100}, time.Unix(200, 0), catalog, 1)
	second.InheritCampaignHistory(first)
	second.InheritCampaignHistory(first) // Assignment must be idempotent.
	want := CareerTotals{Enemies: 12, Bosses: 2, TreasureGoblins: 1, Gold: 123, Gear: 2}
	if second.RecordedTotals() != want {
		t.Fatalf("carry: %+v", second.RecordedTotals())
	}
	second.Stats.Kills = 4
	second.Stats.Bosses = 1
	second.BankedGold = 50
	second.BankedItems = []string{"Shield"}
	second.Status = "complete"
	data, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	third := NewRunAtLevel("third", Build{HP: 100}, time.Unix(300, 0), catalog, 2)
	third.InheritCampaignHistory(&saved)
	want = CareerTotals{Enemies: 16, Bosses: 3, TreasureGoblins: 1, Gold: 173, Gear: 3}
	if third.RecordedTotals() != want {
		t.Fatalf("subsequent carry: %+v", third.RecordedTotals())
	}
	if first.RecordedTotals().Enemies != 12 || second.Stats.Kills != 4 {
		t.Fatal("history inheritance mutated expedition totals")
	}
}

func TestTreasureGoblinRecordCountsOnlyConfirmedDefeats(t *testing.T) {
	r := testRun()
	r.Enemies = []Actor{{Kind: "treasure", HP: 20, MaxHP: 20}, {Kind: "goblin", HP: 20, MaxHP: 20}}
	r.hurtEnemy(0, 5, "hit")
	if r.Stats.TreasureGoblins != 0 {
		t.Fatal("partial damage counted as capture")
	}
	r.hurtEnemy(0, 100, "hit")
	r.hurtEnemy(0, 100, "hit")
	r.hurtEnemy(1, 100, "hit")
	if r.Stats.TreasureGoblins != 1 || r.Stats.Kills != 2 {
		t.Fatalf("incorrect goblin record: %+v", r.Stats)
	}
}

func TestClearResultDistinguishesFirstRepeatAndImprovedRecords(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	r := NewRunAtLevel("first", Build{HP: 100}, time.Unix(100, 0), catalog, 1)
	r.Stats.Seconds = 20
	r.Stats.HitsTaken = 3
	r.Player.HP = 60
	r.finishMissionHistory("completed")
	if r.LastClear == nil || !r.LastClear.First || r.LastClear.Mission != 1 || len(r.LastClear.Records) != 3 {
		t.Fatalf("first clear: %+v", r.LastClear)
	}
	r.finishMissionHistory("completed")
	if !r.LastClear.First {
		t.Fatal("duplicate completion replaced result")
	}
	next := NewRunAtLevel("next", Build{HP: 100}, time.Unix(200, 0), catalog, 1)
	next.InheritCampaignHistory(r)
	next.Stats.Seconds = 20
	next.Stats.HitsTaken = 3
	next.Player.HP = 60
	next.finishMissionHistory("completed")
	if next.LastClear.First || len(next.LastClear.Records) != 0 {
		t.Fatalf("tie claimed new records: %+v", next.LastClear)
	}
	best := NewRunAtLevel("best", Build{HP: 100}, time.Unix(300, 0), catalog, 1)
	best.InheritCampaignHistory(next)
	best.Stats.Seconds = 15
	best.Stats.HitsTaken = 0
	best.Player.HP = 80
	best.Room = 2
	best.Status = "cleared"
	best.FinishCheckpoint("advance", catalog)
	if best.LastClear.First || best.LastClear.Mission != 1 || len(best.LastClear.Records) != 3 || best.Level.ID != 2 {
		t.Fatalf("improved records lost during seamless advance: %+v", best.LastClear)
	}
	data, err := json.Marshal(best)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.LastClear.Mission != 1 || len(restored.LastClear.Records) != 3 {
		t.Fatal("result lost on reload")
	}
	legacy := NewRunAtLevel("legacy", Build{HP: 100}, time.Unix(400, 0), catalog, 1)
	legacy.CompletedLevels = []int{1}
	legacy.finishMissionHistory("completed")
	if legacy.LastClear.First {
		t.Fatal("legacy completion presented as first clear")
	}
	early := NewRunAtLevel("early", Build{HP: 100}, time.Unix(500, 0), catalog, 1)
	early.finishMissionHistory("exited")
	if early.LastClear != nil {
		t.Fatal("early exit presented as clear")
	}
}

func TestPersonalRecordDatesChangeOnlyWhenRecordImproves(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	var previous *Run
	for index, seconds := range []float64{20, 20, 15} {
		r := NewRunAtLevel("dated", Build{HP: 100}, time.Unix(int64(100+100*index), 0), catalog, 1)
		r.InheritCampaignHistory(previous)
		r.Stats.Seconds = seconds
		r.finishMissionHistory("completed")
		h := r.History[1]
		wantTime := int64(100000)
		if index == 2 {
			wantTime = 300000
		}
		if h.BestSecondsAtMS != wantTime || h.BestFinishHPAtMS != 100000 || h.FewestHitsAtMS != 100000 {
			t.Fatalf("incorrect record dates: %+v", h)
		}
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved Run
		if err := json.Unmarshal(data, &saved); err != nil {
			t.Fatal(err)
		}
		previous = &saved
	}
	legacy := NewRunAtLevel("legacy", Build{HP: 100}, time.Unix(500, 0), catalog, 1)
	legacy.History[1] = MissionHistory{Attempts: 1, BestSeconds: 10, BestFinishHP: 100}
	legacy.Stats.Seconds = 20
	legacy.finishMissionHistory("completed")
	if legacy.History[1].BestSecondsAtMS != 0 || legacy.History[1].BestFinishHPAtMS != 0 {
		t.Fatal("invented dates for older records")
	}
}

func TestAttemptHistoryRecordsOutcomesAndRemainsBounded(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	var previous *Run
	for i := 0; i < 52; i++ {
		r := NewRunAtLevel("attempt", Build{HP: 100, Class: "vanguard"}, time.Unix(int64(100+i), 0), catalog, 1)
		r.InheritCampaignHistory(previous)
		r.Stats.Seconds = 12
		r.Stats.HitsTaken = 2
		r.Player.HP = 70
		r.finishMissionHistory("completed")
		r.finishMissionHistory("completed")
		if len(r.AttemptHistory) != min(i+1, 50) {
			t.Fatal("duplicate or unbounded attempt log")
		}
		previous = r
	}
	first := previous.AttemptHistory[0]
	if first.AtMS != 102000 || first.Seconds != 12 || first.Hits == nil || *first.Hits != 2 || first.Class != "vanguard" || first.HP != 70 {
		t.Fatalf("wrong attempt snapshot: %+v", first)
	}
	data, err := json.Marshal(previous)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	early := NewRunAtLevel("early", Build{HP: 100}, time.Unix(200, 0), catalog, 2)
	early.InheritCampaignHistory(&saved)
	early.finishMissionHistory("exited")
	if early.AttemptHistory[49].Outcome != "exited" || saved.AttemptHistory[49].Outcome != "completed" {
		t.Fatal("outcome missing or previous history mutated")
	}
	active := NewRunAtLevel("active", Build{HP: 100}, time.Unix(300, 0), catalog, 2)
	active.InheritCampaignHistory(early)
	next := NewRunAtLevel("next", Build{HP: 100}, time.Unix(400, 0), catalog, 3)
	next.InheritCampaignHistory(active)
	if next.AttemptHistory[49].Outcome != "expired" || next.AttemptHistory[49].AtMS != 400000 {
		t.Fatal("abandoned attempt missing")
	}
	next.MissionStartHits = nil
	next.finishMissionHistory("defeated")
	last := next.AttemptHistory[49]
	if last.Outcome != "defeated" || last.Hits != nil {
		t.Fatal("legacy hit count invented")
	}
}

func TestRoomSplitsExcludeCheckpointWaitingAndSurviveAdvance(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	r := NewRunAtLevel("splits", Build{HP: 100}, time.Unix(100, 0), catalog, 1)
	for room := 0; room < 3; room++ {
		r.Enemies = nil
		r.tick(Input{}, float64(room+1))
		if r.RoomSplits[room] == nil || *r.RoomSplits[room] != float64(room+1) {
			t.Fatalf("room %d split missing: %+v", room, r.RoomSplits)
		}
		r.tick(Input{}, 10)
		if *r.RoomSplits[room] != float64(room+1) {
			t.Fatal("checkpoint waiting changed split")
		}
		if room < 2 {
			r.NextRoom()
		}
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	saved.FinishCheckpoint("advance", catalog)
	if saved.RoomSplits[0] != nil || saved.AttemptHistory[0].Splits[2] == nil || *saved.AttemptHistory[0].Splits[2] != 3 {
		t.Fatal("advance lost completed splits or reused them")
	}
	saved.RoomStartSeconds = nil
	saved.Enemies = nil
	saved.tick(Input{}, 1)
	if saved.RoomSplits[0] != nil {
		t.Fatal("legacy split invented")
	}
}

func TestLoadingResumeGapDoesNotEnterClearTime(t *testing.T) {
	for _, paused := range []bool{false, true} {
		r := NewRunAtLevel("loading", Build{HP: 100}, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
		if paused {
			r.SetPaused(true, time.Unix(100, 0))
		}
		r.SetPaused(false, time.Unix(200, 0))
		if r.Stats.Seconds != 0 {
			t.Fatal("loading entered combat time")
		}
		r.Step(Input{}, time.Unix(200, 100000000))
		r.Room = 2
		r.Status = "cleared"
		r.FinishCheckpoint("bank", content.AbyssMobCatalog())
		if got := r.History[1].BestSeconds; got < 0.099999 || got > 0.100001 {
			t.Fatalf("record included resume gap: %f", got)
		}
	}
}

func TestHealingGuardChipDoesNotCreateFlawlessRoom(t *testing.T) {
	r := NewRunAtLevel("healed-room", Build{HP: 240}, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
	r.Player.Guard = true
	r.Player.Facing = 1
	r.hurtPlayer(10, r.Player.X+20, r.Player.Y)
	if r.Stats.HitsTaken != 1 {
		t.Fatal("guard chip must count as a damaging hit")
	}
	r.healPlayer(r.Player.MaxHP)
	if r.Player.HP != r.Player.MaxHP {
		t.Fatal("test did not restore full health")
	}
	r.Enemies = nil
	r.Level.Rooms[0].Hazards = nil
	r.tick(Input{}, 0)
	if r.Status != "cleared" {
		t.Fatal("room failed to clear")
	}
	if len(r.History[1].FlawlessTiers) != 0 {
		t.Fatal("healing erased the room's damage history")
	}
}
