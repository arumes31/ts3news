package bot

import "ts3news/internal/content"

type shopComparisonView = gearComparison

func shopGearComparison(candidate content.Gear, equipped map[string]content.Gear) shopComparisonView {
	current, occupied := equipped[string(candidate.Slot)]
	return compareGear(candidate, current, occupied)
}
