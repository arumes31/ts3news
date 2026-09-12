package bot

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"time"

	"ts3news/internal/content"
)

const itemComparisonVersion = "item-contributions-v2"

type comparisonItemInput struct {
	Name         string           `json:"name"`
	Slot         content.GearSlot `json:"slot"`
	Rarity       string           `json:"rarity"`
	Temper       int              `json:"temper"`
	Stats        content.Stats    `json:"base_stats"`
	Contribution content.Stats    `json:"contribution"`
	XP           float64          `json:"effective_xp"`
	Regen        float64          `json:"regen_per_second"`
	Sockets      int              `json:"sockets"`
}

// Remove insignificant binary representation noise without erasing small gains.
func preciseXPDelta(after, before float64) float64 {
	delta := (after - before) * 100
	if math.Abs(delta) >= 0.000001 {
		return math.Round(delta*1e9) / 1e9
	}
	return delta
}

func (r gearComparison) JSON() string {
	b, err := json.Marshal(r)
	if err != nil {
		return `{ "unknown": true, "status": "unknown" }`
	}
	return string(b)
}

func enrichGearComparison(r *gearComparison, candidate, current content.Gear, occupied bool, now time.Time) {
	r.Version = itemComparisonVersion
	r.RegenBefore, r.RegenAfter = gearRegenRate(current), gearRegenRate(candidate)
	r.ConditionBefore, r.ConditionAfter = current.ComparisonDurability, candidate.ComparisonDurability
	if occupied {
		r.EquippedRarity = current.Rarity.String()
		r.EquippedTemper = current.Temper
	}
	for _, g := range []content.Gear{current, candidate} {
		r.Inputs = append(r.Inputs, comparisonItemInput{Name: g.Name, Slot: g.Slot, Rarity: g.Rarity.String(), Temper: g.Temper, Stats: g.Stats, Contribution: gearContributionStats(g, now), XP: g.EffectiveXPMultiplier(), Regen: gearRegenRate(g), Sockets: g.Sockets})
	}
	changes := slices.Clone(r.Changes)
	slices.SortStableFunc(changes, func(a, b gearStatChange) int { return cmp.Compare(a.Delta, b.Delta) })
	var reasons []string
	for _, change := range changes {
		if change.Combat && change.Delta < 0 {
			reasons = append(reasons, fmt.Sprintf("Loses %d %s.", -change.Delta, change.Label))
		}
	}
	for _, reason := range r.Reasons {
		if !slices.Contains(reasons, reason) {
			reasons = append(reasons, reason)
		}
	}
	r.Reasons = reasons
}

func gearBrokenInAt(g content.Gear) string {
	if content.IsPetGearSlot(g.Slot) {
		return ""
	}
	t, err := time.Parse(time.RFC3339, g.FoundAt)
	if err != nil {
		return ""
	}
	return t.Add(30 * 24 * time.Hour).Format(time.RFC3339)
}
