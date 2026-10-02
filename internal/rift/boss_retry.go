package rift

import (
	"errors"
	"time"
)

// CanRetryBossEncounter requires a defeated campaign room with its frozen boss lineup.
func (r *Run) CanRetryBossEncounter() bool {
	if r.Practice != nil || r.Level == nil || r.Status != "defeated" || r.Room < 0 || r.Room >= len(Rooms) || len(r.EncounterPlan) != len(Rooms) {
		return false
	}
	for _, actor := range r.EncounterPlan[r.Room] {
		if actor.Kind == "boss" {
			return true
		}
	}
	return false
}

// RetryBossEncounter restores combat resources without restoring lost rewards.
// Mission timing and hit totals include prior failed attempts in this expedition.
func (r *Run) RetryBossEncounter(now time.Time) error {
	if !r.CanRetryBossEncounter() {
		return errors.New("no defeated boss encounter to retry")
	}
	r.Player = NewRunWithCatalog(r.ID, r.Build, now, nil).Player
	r.Status = "fighting"
	r.Gold = 0
	r.Drops = []Drop{}
	r.Events = []Event{}
	r.SkillTimers = map[string]float64{}
	r.Combo, r.Resource = 0, 0
	r.ComboTime, r.Barrier, r.Clock = 0, 0, 0
	r.BarrierSources = nil
	r.jumpAir, r.jumpDist, r.heavyRecovery = 0, 0, 0
	r.Catchup = false
	r.LastMS = max(r.LastMS, now.UnixMilli())
	r.SavedAtMS = r.LastMS
	r.Paused = false
	r.PauseStartedMS = nil
	r.RoomSplits[r.Room] = nil
	missionStart, missionHits := r.MissionStartSeconds, r.MissionStartHits
	r.beginMissionHistory()
	r.MissionStartSeconds, r.MissionStartHits = missionStart, missionHits
	r.spawnRoom()
	r.SetPaused(true, time.UnixMilli(r.LastMS))
	return nil
}
