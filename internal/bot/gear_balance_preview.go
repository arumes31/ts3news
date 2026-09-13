package bot

import (
	"fmt"
	"slices"
	"ts3news/internal/content"
)

// gearRatingMarginalNotes isolates the candidate's rating delta against the
// current character total. It is deliberately not a full build simulation:
// affix, tree, enchantment and set interactions are shown separately.
func gearRatingMarginalNotes(total, before, after content.Stats) []string {
	notes := []string{}
	for _, stat := range []struct {
		name                 string
		total, before, after int
		chance               func(int) float64
	}{
		{"Critical", total.CRT, before.CRT, after.CRT, content.CriticalChance},
		{"Dodge", total.DGE, before.DGE, after.DGE, content.DodgeChance},
	} {
		if stat.before == stat.after {
			continue
		}
		previous, next := stat.chance(stat.total), stat.chance(max(0, stat.total-stat.before+stat.after))
		notes = append(notes, fmt.Sprintf("%s chance from rating: %.2f%% → %.2f%% (%+.2f percentage points).", stat.name, previous, next, next-previous))
	}
	if len(notes) > 0 {
		notes = append(notes, "Rating-only estimate from current permanent character stats; set, tree, affix and temporary combat changes are excluded.")
	}
	return notes
}

func gearPassiveMarginalNotes(candidate content.Gear, equipped map[string]content.Gear) []string {
	if candidate.Unidentified || content.IsPetGearSlot(candidate.Slot) {
		return nil
	}
	before, after := []content.ItemEffect{}, []content.ItemEffect{}
	for slot, gear := range equipped {
		if gear.Unidentified || content.IsPetGearSlot(gear.Slot) {
			continue
		}
		before = append(before, gear.Effects()...)
		if slot != string(candidate.Slot) {
			after = append(after, gear.Effects()...)
		}
	}
	after = append(after, candidate.Effects()...)
	notes := []string{}
	for _, effect := range []content.ItemEffect{content.EffectExecutioner, content.EffectThorns, content.EffectTreasureHunter, content.EffectLucky, content.EffectQuick, content.EffectBulwark, content.EffectFocused, content.EffectRadiant, content.EffectVampiric, content.EffectBerserk, content.EffectFragile} {
		oldCount, newCount := 0, 0
		for _, e := range before {
			if e == effect {
				oldCount++
			}
		}
		for _, e := range after {
			if e == effect {
				newCount++
			}
		}
		if oldCount == newCount {
			continue
		}
		oldBonus, newBonus := content.PassiveEffectBonus(before, effect), content.PassiveEffectBonus(after, effect)
		notes = append(notes, fmt.Sprintf("%s equipped-gear bonus: %.2f%% → %.2f%% (%+.2f percentage points); duplicate copies diminish and the aggregate is capped.", effect, oldBonus*100, newBonus*100, (newBonus-oldBonus)*100))
	}
	for _, effect := range []content.ItemEffect{content.EffectParry, content.EffectStealth, content.EffectPhoenix, content.EffectCleanse, content.EffectSteady} {
		if slices.Contains(candidate.Effects(), effect) && slices.Contains(before, effect) {
			notes = append(notes, fmt.Sprintf("%s is already present; duplicate copies do not stack.", effect))
		}
	}
	return notes
}

// Template methods expose the canonical permanent character chances.
func (u webUser) CriticalChance() float64 { return content.CriticalChance(u.Stats.CRT) }
func (u webUser) DodgeChance() float64    { return content.DodgeChance(u.Stats.DGE) }
