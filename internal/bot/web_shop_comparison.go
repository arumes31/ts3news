package bot

import "ts3news/internal/content"

type shopComparisonView = gearComparison

func shopGearComparison(candidate content.Gear, equipped map[string]content.Gear) shopComparisonView {
	if candidate.ComparisonDurability == nil {
		value := candidate.MaxDurability
		candidate.ComparisonDurability = &value
	}
	current, occupied := equipped[string(candidate.Slot)]
	comparison := compareGear(candidate, current, occupied)
	if !comparison.Unknown {
		comparison.Reasons = append(comparison.Reasons, gearPassiveMarginalNotes(candidate, equipped)...)
	}
	return comparison
}
