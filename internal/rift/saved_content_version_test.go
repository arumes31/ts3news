package rift

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestSavedContentVersionAcrossFullExpedition(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	run := NewRunAtLevel("content-version", Build{HP: 300}, time.Unix(100, 0), catalog, 1)
	labelFormat := regexp.MustCompile(`^level-v1:[0-9a-f]{64}$`)
	previous := ""
	for mission := 1; mission <= LevelCount; mission++ {
		version := run.MissionDefinition
		if !labelFormat.MatchString(version) || version == previous || run.Level.ID != mission {
			t.Fatalf("mission %d has invalid/reused version %q", mission, version)
		}
		for tier := 0; tier < len(Rooms); tier++ {
			if run.Room != tier || run.MissionDefinition != version {
				t.Fatalf("tier transition changed mission %d version", mission)
			}
			raw, err := json.Marshal(run)
			if err != nil {
				t.Fatal(err)
			}
			var saved Run
			if err = json.Unmarshal(raw, &saved); err != nil {
				t.Fatal(err)
			}
			if saved.MissionDefinition != version || saved.History[mission].Definition != version {
				t.Fatalf("save lost mission %d version", mission)
			}
			run = &saved
			run.Status = "cleared"
			run.FinishCheckpoint("advance", catalog)
		}
		last := run.AttemptHistory[len(run.AttemptHistory)-1]
		if last.Mission != mission || last.Definition != version {
			t.Fatalf("completion lost mission %d version", mission)
		}
		previous = version
	}
	if run.Status != "complete" {
		t.Fatalf("final status %q", run.Status)
	}
}
