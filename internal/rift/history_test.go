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
