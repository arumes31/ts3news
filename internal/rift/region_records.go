package rift

import "maps"

type RegionRecord struct {
	BestSeconds float64 `json:"best_seconds"`
	AtMS        int64   `json:"at_ms"`
}
type RegionAttempt struct {
	Region      int     `json:"region"`
	NextMission int     `json:"next_mission"`
	Seconds     float64 `json:"seconds"`
}

func (r *Run) recordRegionTime(elapsed float64) {
	if r.Practice != nil || r.Level == nil || elapsed <= 0 {
		r.RegionAttempt = nil
		return
	}
	id := r.Level.ID
	region := (id - 1) / 10
	if id%10 == 1 {
		r.RegionAttempt = &RegionAttempt{Region: region, NextMission: id}
	}
	attempt := r.RegionAttempt
	if attempt == nil || attempt.Region != region || attempt.NextMission != id {
		r.RegionAttempt = nil
		return
	}
	attempt.Seconds += elapsed
	attempt.NextMission++
	if id%10 != 0 {
		return
	}
	if r.RegionRecords == nil {
		r.RegionRecords = map[int]RegionRecord{}
	}
	old := r.RegionRecords[region]
	if old.BestSeconds == 0 || attempt.Seconds < old.BestSeconds {
		r.RegionRecords[region] = RegionRecord{BestSeconds: attempt.Seconds, AtMS: r.LastMS}
	}
	r.RegionAttempt = nil
}
func (r *Run) inheritRegionRecords(previous *Run) {
	r.RegionRecords = maps.Clone(previous.RegionRecords)
}
