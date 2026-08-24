package bot

import (
	"errors"

	"ts3news/internal/content"
)

func selectFusionSurvivor(items []content.Gear, avoidDuplicates bool, remainingDuplicates map[string]bool) (content.Gear, error) {
	if len(items) == 0 {
		return content.Gear{}, errors.New("no fusion inputs")
	}
	bestIndex := -1
	for index, item := range items {
		if avoidDuplicates && remainingDuplicates[item.ID] {
			continue
		}
		if bestIndex < 0 || item.CombatRating() > items[bestIndex].CombatRating() {
			bestIndex = index
		}
	}
	if bestIndex < 0 {
		return content.Gear{}, errors.New("every possible fusion result is already owned")
	}
	return items[bestIndex], nil
}
