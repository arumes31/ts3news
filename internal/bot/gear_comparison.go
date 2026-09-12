package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"time"

	"ts3news/internal/content"
)

const comparisonUnavailableID = "\x00comparison-unavailable"

// Recommendation reads must preserve decoding errors instead of substituting a catalog roll.
func (b *Bot) makeComparisonGear(id string, data sql.NullString) (content.Gear, bool) {
	if data.Valid && data.String != "" {
		var check content.Gear
		if json.Unmarshal([]byte(data.String), &check) != nil {
			return content.Gear{}, false
		}
	}
	return b.makeGear(id, data)
}

func gearContributionStats(g content.Gear, now time.Time) content.Stats {
	if content.IsPetGearSlot(g.Slot) {
		return g.Stats
	}
	return g.Stats.Add(sentimentalValueBonus(g, now))
}

// A failed or partial equipment read must never look like an empty loadout.
func (b *Bot) getEquippedComparisonItems(uid string) map[content.GearSlot]content.Gear {
	unknown := func() map[content.GearSlot]content.Gear {
		out := map[content.GearSlot]content.Gear{}
		for _, slot := range content.AllSlots {
			out[slot] = content.Gear{ID: comparisonUnavailableID, Slot: slot, Unidentified: true}
		}
		return out
	}
	rows, err := b.DB.Query("SELECT slot, gear_id, item_data FROM user_gear WHERE client_uid = $1", uid)
	if err != nil {
		return unknown()
	}
	defer func() { _ = rows.Close() }()
	out := map[content.GearSlot]content.Gear{}
	for rows.Next() {
		var slot, id string
		var data sql.NullString
		if rows.Scan(&slot, &id, &data) != nil {
			return unknown()
		}
		g, ok := b.makeComparisonGear(id, data)
		if !ok || g.Slot != content.GearSlot(slot) || !slices.Contains(content.AllSlots, g.Slot) {
			return unknown()
		}
		out[g.Slot] = g
	}
	if rows.Err() != nil {
		return unknown()
	}
	return out
}

type gearStatChange struct {
	Code        string  `json:"code"`
	Label       string  `json:"label"`
	Description string  `json:"description"`
	Combat      bool    `json:"combat"`
	Before      int     `json:"before"`
	After       int     `json:"after"`
	Delta       int     `json:"delta"`
	Percent     float64 `json:"percent"`
	HasPercent  bool    `json:"has_percent"`
}

type gearComparison struct {
	Unknown         bool                 `json:"unknown"`
	EmptySlot       bool                 `json:"empty_slot"`
	IsUpgrade       bool                 `json:"is_upgrade"`
	Status          string               `json:"status"`
	Reasons         []string             `json:"reasons"`
	EquippedName    string               `json:"equipped_name"`
	CRDelta         float64              `json:"cr_delta"`
	ScoreDelta      int                  `json:"score_delta"`
	Power           float64              `json:"power"`
	PowerDelta      float64              `json:"power_delta"`
	XPBonusDelta    int                  `json:"xp_bonus_delta"`
	XP              content.GearXPDetail `json:"xp"`
	Gains           int                  `json:"gains"`
	Losses          int                  `json:"losses"`
	RegenDelta      float64              `json:"regen_delta"`
	DurabilityDelta int                  `json:"durability_delta"`
	SocketDelta     int                  `json:"socket_delta"`
	Stats           []statKV             `json:"stats"`
	Changes         []gearStatChange     `json:"changes"`
	AddedSpecials   []itemSpecialView    `json:"added_specials"`
	RemovedSpecials []itemSpecialView    `json:"removed_specials"`
}

func gearRegenRate(g content.Gear) float64 {
	if g.RegenAmount <= 0 || g.RegenIntervalSec <= 0 {
		return 0
	}
	return float64(g.RegenAmount) / float64(g.RegenIntervalSec)
}

func equalGearGems(a, b []string) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

// An empty slot has no effect to preserve. Known beneficial affixes can be
// accepted, while durability tradeoffs and unrecognized effects need a choice.
func gearEffectNeedsEmptySlotReview(effect content.ItemEffect) bool {
	switch effect {
	case content.EffectNone, content.EffectThorns, content.EffectVampiric, content.EffectBerserk,
		content.EffectLucky, content.EffectTreasureHunter, content.EffectQuick, content.EffectBulwark,
		content.EffectRadiant, content.EffectSteady, content.EffectMindControl, content.EffectRegenStack,
		content.EffectPhoenix, content.EffectStealth, content.EffectParry, content.EffectCleanse,
		content.EffectExecutioner, content.EffectFocused:
		return false
	default:
		return true
	}
}

func compareGear(candidate, current content.Gear, occupied bool) gearComparison {
	unknown := func(reason string) gearComparison {
		return gearComparison{Unknown: true, Status: "unknown", Reasons: []string{reason}}
	}
	if occupied && current.ID == comparisonUnavailableID {
		return unknown("Equipped items could not be loaded reliably. Refresh before choosing an upgrade.")
	}
	if candidate.Unidentified || (occupied && current.Unidentified) {
		return unknown("Identify the item before comparing its hidden roll.")
	}
	if !slices.Contains(content.AllSlots, candidate.Slot) || (occupied && current.Slot != candidate.Slot) {
		return unknown("Equipment slot is invalid or does not match the equipped item.")
	}
	if !candidate.XPDetail().Valid || (occupied && !current.XPDetail().Valid) {
		return unknown("Invalid XP data prevents a reliable comparison.")
	}
	if candidate.RegenAmount < 0 || (candidate.RegenAmount > 0 && candidate.RegenIntervalSec <= 0) ||
		(occupied && (current.RegenAmount < 0 || (current.RegenAmount > 0 && current.RegenIntervalSec <= 0))) {
		return unknown("Invalid regeneration data prevents a reliable comparison.")
	}
	if !occupied {
		current = content.Gear{Slot: candidate.Slot, XPMultiplier: 1}
	}
	now := time.Now()
	beforeStats, afterStats := gearContributionStats(current, now), gearContributionStats(candidate, now)
	r := gearComparison{EmptySlot: !occupied, EquippedName: current.Name, Power: afterStats.Power(), XP: candidate.XPDetail(),
		CRDelta: candidate.CombatRating() - current.CombatRating(), ScoreDelta: candidate.Stats.Score() - current.Stats.Score(),
		PowerDelta:   math.Round((afterStats.Power()-beforeStats.Power())*10) / 10,
		XPBonusDelta: int(math.Round((candidate.EffectiveXPMultiplier() - current.EffectiveXPMultiplier()) * 100)),
		RegenDelta:   gearRegenRate(candidate) - gearRegenRate(current), DurabilityDelta: candidate.MaxDurability - current.MaxDurability, SocketDelta: candidate.Sockets - current.Sockets}
	before, after := beforeStats.Details(), afterStats.Details()
	if !content.IsPetGearSlot(candidate.Slot) && (current.BrokenIn(now) || candidate.BrokenIn(now)) {
		r.Reasons = append(r.Reasons, "Stat comparison includes the current broken-in bonus.")
	}
	for i, stat := range after {
		if stat.Value == before[i].Value {
			continue
		}
		change := gearStatChange{Code: stat.Code, Label: stat.Label, Description: stat.Description, Combat: stat.Combat, Before: before[i].Value, After: stat.Value, Delta: stat.Value - before[i].Value}
		if change.Before != 0 {
			change.HasPercent = true
			change.Percent = float64(change.Delta) / math.Abs(float64(change.Before)) * 100
		}
		r.Changes = append(r.Changes, change)
		r.Stats = append(r.Stats, statKV{Label: stat.Label, Value: change.Delta})
		if stat.Combat {
			if change.Delta > 0 {
				r.Gains++
			} else {
				r.Losses++
				r.Reasons = append(r.Reasons, fmt.Sprintf("Loses %d %s.", -change.Delta, stat.Label))
			}
		}
	}
	gains, losses := r.Gains, r.Losses
	measure := func(delta float64, gain, loss string) {
		if delta > 1e-9 {
			gains++
			if gain != "" {
				r.Reasons = append(r.Reasons, gain)
			}
		}
		if delta < -1e-9 {
			losses++
			r.Reasons = append(r.Reasons, loss)
		}
	}
	measure(candidate.EffectiveXPMultiplier()-current.EffectiveXPMultiplier(), "Effective XP improves.", "Effective XP decreases.")
	measure(r.RegenDelta, "Regeneration per second improves.", "Regeneration per second decreases.")
	measure(float64(r.DurabilityDelta), "Maximum durability improves.", "Maximum durability decreases.")
	measure(float64(r.SocketDelta), "Socket capacity improves.", "Socket capacity decreases.")
	contextual := false
	context := func(changed bool, reason string) {
		if changed {
			contextual = true
			r.Reasons = append(r.Reasons, reason)
		}
	}
	if !occupied {
		context(candidate.Cursed || candidate.Eldritch || candidate.Doomed, "Cursed, eldritch, or doomed items require manual choice, even in an empty slot.")
	}
	if occupied {
		context(candidate.Rune != current.Rune, "Rune changes require manual comparison.")
		context(candidate.Element != current.Element, "Element changes depend on the encounter and build.")
		context(candidate.EffectiveSetID() != current.EffectiveSetID(), "Set membership changes; check the equipped set bonus.")
		context(!equalGearGems(candidate.Gemstones, current.Gemstones), "Fitted gems change; check gem tiers and resonance.")
		context(candidate.Cursed != current.Cursed || candidate.Eldritch != current.Eldritch || candidate.Doomed != current.Doomed, "Cursed, eldritch, or doomed properties change.")
		context(candidate.Attuned != current.Attuned, "Attunement changes; preserve bound progression or choose manually.")
		if current.Insured && !candidate.Insured {
			losses++
			r.Reasons = append(r.Reasons, "Loses item insurance.")
		}
		if !current.Insured && candidate.Insured {
			gains++
			r.Reasons = append(r.Reasons, "Gains item insurance.")
		}
	}
	effects := func(g content.Gear) map[content.ItemEffect]bool {
		out := map[content.ItemEffect]bool{}
		for _, e := range append([]content.ItemEffect{g.Special}, g.BonusEffects...) {
			if e != content.EffectNone {
				out[e] = true
			}
		}
		return out
	}
	oldEffects, newEffects := effects(current), effects(candidate)
	// Iterate the stable displayed order, while identity checks use effect IDs.
	appendEffects := func(g content.Gear, other map[content.ItemEffect]bool) []itemSpecialView {
		var out []itemSpecialView
		seen := map[content.ItemEffect]bool{}
		for _, e := range append([]content.ItemEffect{g.Special}, g.BonusEffects...) {
			if e == content.EffectNone || seen[e] || other[e] {
				continue
			}
			seen[e] = true
			out = append(out, gearSpecialViews(content.Gear{Special: e})...)
		}
		return out
	}
	r.AddedSpecials = appendEffects(candidate, oldEffects)
	r.RemovedSpecials = appendEffects(current, newEffects)
	effectReview := occupied && (len(r.AddedSpecials) > 0 || len(r.RemovedSpecials) > 0)
	if !occupied {
		for effect := range newEffects {
			effectReview = effectReview || gearEffectNeedsEmptySlotReview(effect)
		}
	}
	context(effectReview, "Special effects change; compare their combat benefits and costs.")
	switch {
	case contextual || (gains > 0 && losses > 0):
		r.Status = "tradeoff"
	case losses > 0:
		r.Status = "downgrade"
	case gains > 0 || !occupied:
		r.Status = "upgrade"
		r.IsUpgrade = true
	default:
		r.Status = "equivalent"
	}
	if len(r.Reasons) == 0 {
		switch r.Status {
		case "upgrade":
			r.Reasons = []string{"Improves item contributions without a measured loss."}
		case "equivalent":
			r.Reasons = []string{"No effective combat-stat or XP improvement; rarity alone does not make an upgrade."}
		}
	}
	return r
}
