package bot

import (
	"errors"
	"time"
	"ts3news/internal/content"
	"ts3news/internal/rift"
)

// buildLargeRiftReceipt provides the same synthetic full-campaign receipt for
// codec regressions and the isolated browser fixture. It grants no real loot.
func buildLargeRiftReceipt(build rift.Build, now time.Time) (*rift.Run, error) {
	var elite content.Mob
	for _, mob := range content.AbyssMobCatalog() {
		if mob.Type == content.MobElite {
			elite = mob
			break
		}
	}
	if elite.Name == "" {
		return nil, errors.New("missing elite fixture")
	}
	catalog := []content.Mob{elite}
	run := rift.NewRunAtLevel("large-receipt", build, now, catalog, 1)
	for mission := 1; mission <= rift.LevelCount; mission++ {
		for room := range rift.Rooms {
			var gear content.Gear
			for _, candidate := range content.AbyssGearCatalog() {
				if candidate.Rarity <= rift.LootRarityCap(room) && len(candidate.Name) > len(gear.Name) {
					gear = candidate
				}
			}
			gear.FoundBoss = riftGearOrigin(run)
			gear.FoundAt = now.Format(time.RFC3339)
			for _, enemy := range run.Enemies {
				drop := rift.Drop{ID: enemy.ID, Mission: mission, Tier: room + 1, Gear: &gear, Collected: true, Banked: true}
				run.BankedItems = append(run.BankedItems, gear.Name)
				run.BankedLoot = append(run.BankedLoot, drop.LootReceipt())
			}
			run.Status = "cleared"
			run.Stats.Seconds += 10
			run.FinishCheckpoint("advance", catalog)
		}
	}
	return run, nil
}
