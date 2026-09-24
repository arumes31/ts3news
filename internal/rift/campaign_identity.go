package rift

import (
	"fmt"
	"strings"
)

// Campaign content is compiled into the binary. Reject invalid authoring at
// startup, before any request can select a mission by its slice position.
// This deliberately does not validate or rewrite historical saved expeditions.
func init() {
	if err := validateCampaignIdentity(Campaign()); err != nil {
		panic(fmt.Sprintf("invalid Brawl campaign content: %v", err))
	}
}

func validateCampaignIdentity(levels []Level) error {
	if len(levels) != LevelCount {
		return fmt.Errorf("expected %d missions, got %d", LevelCount, len(levels))
	}
	ids := make(map[int]int, len(levels))
	names := make(map[string]int, len(levels))
	for index, level := range levels {
		if level.ID < 1 || level.ID > LevelCount {
			return fmt.Errorf("mission ID %d at position %d outside 1..%d", level.ID, index+1, LevelCount)
		}
		if previous, exists := ids[level.ID]; exists {
			return fmt.Errorf("duplicate mission ID %d at positions %d and %d", level.ID, previous+1, index+1)
		}
		ids[level.ID] = index
		name := strings.ToLower(strings.TrimSpace(level.Name))
		if name == "" {
			return fmt.Errorf("empty mission name for ID %d", level.ID)
		}
		if previous, exists := names[name]; exists {
			return fmt.Errorf("duplicate mission name %q for IDs %d and %d", level.Name, previous, level.ID)
		}
		names[name] = level.ID
	}
	for index, level := range levels {
		if level.ID != index+1 {
			return fmt.Errorf("position %d has mission ID %d; IDs must match selection order", index+1, level.ID)
		}
	}
	return nil
}
