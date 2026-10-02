//go:build e2e

package bot

import (
	"math"
	"strings"
	"time"
	"ts3news/internal/rift"
)

// Select the first authored example so future campaign hazard kinds can be
// inspected without maintaining a separate fixture roster or copied timings.
func riftHazardFixture(scenario string, build rift.Build) *rift.Run {
	if !strings.HasPrefix(scenario, "hazard-") {
		return nil
	}
	kind := strings.TrimPrefix(scenario, "hazard-")
	for _, level := range rift.Campaign() {
		for room, arena := range level.Rooms {
			for _, hazard := range arena.Hazards {
				if hazard.Kind != kind || hazard.Disabled {
					continue
				}
				now := time.Now()
				run := rift.NewRunAtLevel(scenario, build, now, riftMobCatalog(now), level.ID)
				for run.Room < room {
					run.Status = "cleared"
					run.NextRoom()
				}
				run.Level.Rooms[room].Hazards = []rift.Hazard{hazard}
				run.Clock = math.Mod(1.2+hazard.Duration/2-hazard.Offset, hazard.Period)
				if run.Clock < 0 {
					run.Clock += hazard.Period
				}
				run.Epoch = "fixture"
				run.SetPaused(true, now)
				return run
			}
		}
	}
	return nil
}
