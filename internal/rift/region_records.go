package rift

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"strings"
)

type RegionRecord struct {
	Definition  string  `json:"definition,omitempty"`
	BestSeconds float64 `json:"best_seconds"`
	AtMS        int64   `json:"at_ms"`
}
type RegionAttempt struct {
	Definitions []string `json:"definitions,omitempty"`
	Region      int      `json:"region"`
	NextMission int      `json:"next_mission"`
	Seconds     float64  `json:"seconds"`
}

func (r *Run) recordRegionTime(elapsed float64) {
	if r.Practice != nil || r.Level == nil || elapsed <= 0 || r.MissionDefinition == "" {
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
	attempt.Definitions = append(attempt.Definitions, r.MissionDefinition)
	attempt.Seconds += elapsed
	attempt.NextMission++
	if id%10 != 0 {
		return
	}
	if r.RegionRecords == nil {
		r.RegionRecords = map[int]RegionRecord{}
	}
	if len(attempt.Definitions) != 10 {
		r.RegionAttempt = nil
		return
	}
	sum := sha256.Sum256([]byte(strings.Join(attempt.Definitions, "\n")))
	definition := "region-v1:" + hex.EncodeToString(sum[:])
	if r.RegionVersions == nil {
		r.RegionVersions = map[int]map[string]RegionRecord{}
	}
	versions := r.RegionVersions[region]
	if versions == nil {
		versions = map[string]RegionRecord{}
		r.RegionVersions[region] = versions
	}
	// Preserve the previous displayed record, including unversioned legacy results.
	displayed := r.RegionRecords[region]
	if displayed.BestSeconds > 0 {
		existing, ok := versions[displayed.Definition]
		if !ok || displayed.BestSeconds < existing.BestSeconds {
			versions[displayed.Definition] = displayed
		}
	}
	old, exists := versions[definition]
	if !exists || attempt.Seconds < old.BestSeconds {
		old = RegionRecord{Definition: definition, BestSeconds: attempt.Seconds, AtMS: r.LastMS}
		versions[definition] = old
	}
	r.RegionRecords[region] = old
	r.RegionAttempt = nil
}
func (r *Run) inheritRegionRecords(previous *Run) {
	r.RegionRecords = maps.Clone(previous.RegionRecords)
	r.RegionVersions = make(map[int]map[string]RegionRecord, len(previous.RegionVersions))
	for region, versions := range previous.RegionVersions {
		r.RegionVersions[region] = maps.Clone(versions)
	}
}
