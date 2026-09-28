package bot

import (
	"errors"
	"strings"

	"ts3news/internal/rift"
)

func validateRiftLevelMetadata(run *rift.Run) error {
	level := run.Level
	// Original expeditions predate authored campaign metadata.
	if level == nil {
		return nil
	}
	invalid := errors.New("invalid rift snapshot level metadata")
	if level.ID < 1 || level.ID > rift.LevelCount || level.Region != (level.ID-1)/10 || len(level.Rooms) != len(rift.Rooms) {
		return invalid
	}
	if strings.TrimSpace(level.Name) == "" || strings.TrimSpace(level.RegionName) == "" || strings.TrimSpace(level.Difficulty) == "" {
		return invalid
	}
	for _, room := range level.Rooms {
		if strings.TrimSpace(room.Name) == "" || !room.ValidBridges() {
			return invalid
		}
	}
	// Do not compare saved labels, geometry, or definition hashes with today's
	// campaign: a content update must not alter an expedition already in progress.
	return nil
}
