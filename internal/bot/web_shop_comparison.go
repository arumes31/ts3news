package bot

import (
	"math"

	"ts3news/internal/content"
)

type shopComparisonView struct {
	Unknown         bool
	EquippedName    string
	CRDelta         float64
	ScoreDelta      int
	XPBonusDelta    int
	Stats           []statKV
	AddedSpecials   []itemSpecialView
	RemovedSpecials []itemSpecialView
}

func shopGearComparison(candidate content.Gear, equipped map[string]content.Gear) shopComparisonView {
	current, ok := equipped[string(candidate.Slot)]
	if !ok {
		return shopComparisonView{CRDelta: candidate.CombatRating(), ScoreDelta: candidate.Stats.Score()}
	}
	if current.Unidentified {
		return shopComparisonView{Unknown: true}
	}
	comparison := shopComparisonView{
		EquippedName: current.Name,
		CRDelta:      candidate.CombatRating() - current.CombatRating(),
		ScoreDelta:   candidate.Stats.Score() - current.Stats.Score(),
		XPBonusDelta: int(math.Round((candidate.XPMultiplier - current.XPMultiplier) * 100)),
	}
	oldStats := map[string]int{}
	for _, stat := range gearStatList(current.Stats) {
		oldStats[stat.Label] = stat.Value
	}
	for _, stat := range gearStatList(candidate.Stats) {
		delta := stat.Value - oldStats[stat.Label]
		if delta != 0 {
			comparison.Stats = append(comparison.Stats, statKV{Label: stat.Label, Value: delta})
		}
		delete(oldStats, stat.Label)
	}
	// Keep the established stat order, including stats that the candidate loses.
	for _, stat := range gearStatList(current.Stats) {
		if value, exists := oldStats[stat.Label]; exists {
			comparison.Stats = append(comparison.Stats, statKV{Label: stat.Label, Value: -value})
		}
	}
	newSpecials, oldSpecials := gearSpecialViews(candidate), gearSpecialViews(current)
	contains := func(specials []itemSpecialView, name string) bool {
		for _, special := range specials {
			if special.Name == name {
				return true
			}
		}
		return false
	}
	for _, special := range newSpecials {
		if !contains(oldSpecials, special.Name) {
			comparison.AddedSpecials = append(comparison.AddedSpecials, special)
		}
	}
	for _, special := range oldSpecials {
		if !contains(newSpecials, special.Name) {
			comparison.RemovedSpecials = append(comparison.RemovedSpecials, special)
		}
	}
	return comparison
}
