package rift

import (
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestFinisherObjectiveRequiresBuilderAndFinisher(t *testing.T) {
	for _, roles := range [][]string{{}, {"builder"}, {"finisher"}, {"builder", "finisher"}} {
		build := testRun().Build
		for _, role := range roles {
			build.Signatures = append(build.Signatures, Skill{Role: role})
		}
		found := false
		for _, entry := range ObjectiveOptions(build) {
			found = found || entry.ID == "finisher"
		}
		if found != (len(roles) == 2) {
			t.Fatalf("wrong eligibility for %v", roles)
		}
	}
}

func TestFinisherObjectiveTracksChargedCastsAcrossSubclasses(t *testing.T) {
	for _, class := range []string{"vanguard", "berserker", "marksman", "beastmaster", "elementalist", "chronomancer", "oracle", "geomancer", "bloodblade", "voidwalker", "runesmith", "alchemist"} {
		t.Run(class, func(t *testing.T) {
			build := testRun().Build
			build.Class = class
			build.Signatures = []Skill{{ID: "builder", Role: "builder", Kind: "shield", Cost: 1}, {ID: "finisher", Role: "finisher", Kind: "slash", Cost: 1, Power: 2}}
			r := NewRunAtLevel("finisher-objective", build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
			r.Stats.ChargedFinishers = 2
			r.setLevel(2, content.AbyssMobCatalog())
			entry := &r.Objectives.Entries[len(r.Objectives.Entries)-1]
			r.cast("finisher")
			r.UpdateObjectives()
			if entry.Current != 0 {
				t.Fatal("empty finisher counted")
			}
			r.Player.Cooldown = 0
			r.cast("builder")
			r.Player.Cooldown = 0
			r.cast("finisher")
			r.UpdateObjectives()
			if entry.ID != "finisher" || entry.Current != 1 || entry.Status != "active" {
				t.Fatal("charged cast not tracked or completed early")
			}
			r.Room = 2
			r.Status = "cleared"
			r.FinishCheckpoint("bank", nil)
			if entry.Status != "complete" || r.ObjectiveHistory["Wayfarer"]["finisher"] != 1 {
				t.Fatal("finisher completion not banked")
			}
		})
	}
}
