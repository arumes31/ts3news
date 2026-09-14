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
