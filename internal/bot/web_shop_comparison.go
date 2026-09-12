package bot

import "ts3news/internal/content"

type shopComparisonView = gearComparison

func shopGearComparison(candidate content.Gear, equipped map[string]content.Gear) shopComparisonView {
	if candidate.ComparisonDurability == nil {
		value := candidate.MaxDurability
		candidate.ComparisonDurability = &value
	}
	current, occupied := equipped[string(candidate.Slot)]
	return compareGear(candidate, current, occupied)
}
