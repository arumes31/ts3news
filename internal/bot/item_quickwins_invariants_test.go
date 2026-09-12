package bot

import (
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"ts3news/internal/content"
)

func TestItemQuickwinsIdenticalEquipmentIsEquivalent(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	gear := content.Gear{Slot: content.SlotHead, Rarity: content.RarityRare, XPMultiplier: 1.012345,
		Stats:   content.Stats{HP: -100, MNA: 50, STR: 1000, DEF: 70, CHA: 3},
		FoundAt: now.Add(-31 * 24 * time.Hour).Format(time.RFC3339),
		Sockets: 2, Gemstones: []string{"Ruby", "Sapphire"}, MaxDurability: 90,
		Insured: true, Special: content.EffectQuick, Rune: "Fire", SetID: "warden"}
	got := compareGearAt(gear, gear, true, now)
	if got.Unknown || got.IsUpgrade || got.Status != "equivalent" || len(got.Changes) != 0 {
		t.Fatalf("identical equipment is not equivalent: %+v", got)
	}
	if got.CRDelta != 0 || got.PowerDelta != 0 || got.ScoreDelta != 0 || got.XPBonusDelta != 0 || got.Gains != 0 || got.Losses != 0 {
		t.Fatalf("identical equipment has nonzero deltas: %+v", got)
	}
	wantCodes := []string{"HP", "MNA", "STR", "DEF", "SPD", "CRT", "DGE", "LCK", "INT", "STA", "CHA", "STN", "SHN", "HGR"}
	var codes []string
	for _, row := range got.AllStats {
		codes = append(codes, row.Code)
		if row.Before != row.After || row.Delta != 0 || row.Percent != 0 {
			t.Errorf("unchanged row has a delta: %+v", row)
		}
	}
	if !slices.Equal(codes, wantCodes) {
		t.Fatalf("all-stat order = %v, want %v", codes, wantCodes)
	}
}

func TestItemQuickwinsReversingComparisonNegatesNumericDeltas(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	a := content.Gear{Slot: content.SlotHead, Rarity: content.RarityRare, XPMultiplier: 1.01,
		Stats:   content.Stats{HP: -200, MNA: 30, STR: 1000, DEF: 70, CHA: 3},
		FoundAt: now.Add(-31 * 24 * time.Hour).Format(time.RFC3339),
		Sockets: 1, MaxDurability: 50, RegenAmount: 1, RegenIntervalSec: 3}
	b := a
	b.Stats = content.Stats{HP: -170, MNA: 20, STR: 1011, DEF: 80, CHA: 1}
	b.FoundAt, b.XPMultiplier = "", 1.013456
	b.Sockets, b.MaxDurability, b.RegenAmount = 2, 60, 2
	forward, reverse := compareGearAt(b, a, true, now), compareGearAt(a, b, true, now)
	if forward.Status != "tradeoff" || reverse.Status != "tradeoff" {
		t.Fatalf("mixed changes lost tradeoff classification: %s / %s", forward.Status, reverse.Status)
	}
	for name, pair := range map[string][2]float64{
		"CR": {forward.CRDelta, reverse.CRDelta}, "power": {forward.PowerDelta, reverse.PowerDelta},
		"score":      {float64(forward.ScoreDelta), float64(reverse.ScoreDelta)},
		"XP":         {float64(forward.XPBonusDelta), float64(reverse.XPBonusDelta)},
		"regen":      {forward.RegenDelta, reverse.RegenDelta},
		"durability": {float64(forward.DurabilityDelta), float64(reverse.DurabilityDelta)},
		"sockets":    {float64(forward.SocketDelta), float64(reverse.SocketDelta)},
	} {
		if math.Abs(pair[0]+pair[1]) > 1e-12 {
			t.Errorf("%s does not negate on reversal: %v", name, pair)
		}
	}
	if len(forward.AllStats) != 14 || len(reverse.AllStats) != 14 || len(forward.Changes) != len(reverse.Changes) {
		t.Fatalf("reversal changed row coverage: %+v / %+v", forward, reverse)
	}
	for i, row := range forward.AllStats {
		back := reverse.AllStats[i]
		if row.Code != back.Code || row.Before != back.After || row.After != back.Before || row.Delta != -back.Delta {
			t.Errorf("stat does not reverse: %+v / %+v", row, back)
		}
	}
}

func TestItemQuickwinsBrokenInExactThirtyDayBoundary(t *testing.T) {
	found := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	boundary := found.Add(30 * 24 * time.Hour)
	old := content.Gear{Slot: content.SlotHead, Stats: content.Stats{STR: 1000}, FoundAt: found.Format(time.RFC3339)}
	fresh := content.Gear{Slot: content.SlotHead, Stats: content.Stats{STR: 1001}}
	for _, tc := range []struct {
		name          string
		now           time.Time
		broken        bool
		status        string
		before, delta int
	}{
		{"one_nanosecond_before", boundary.Add(-time.Nanosecond), false, "upgrade", 1000, 1},
		{"exact_boundary", boundary, true, "downgrade", 1010, -9},
		{"one_nanosecond_after", boundary.Add(time.Nanosecond), true, "downgrade", 1010, -9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := old.BrokenIn(tc.now); got != tc.broken {
				t.Fatalf("BrokenIn = %v, want %v", got, tc.broken)
			}
			got := compareGearAt(fresh, old, true, tc.now)
			if got.Status != tc.status || len(got.Changes) != 1 || got.Changes[0].Before != tc.before || got.Changes[0].Delta != tc.delta {
				t.Fatalf("boundary comparison: %+v", got)
			}
		})
	}
}

func TestItemQuickwinsPercentagesHandleNegativeAndLargeStats(t *testing.T) {
	large := int(^uint(0)>>1) / 4
	for _, tc := range []struct {
		name          string
		before, after int
		percent       float64
		hasPercent    bool
	}{
		{"negative_improves", -100, -90, 10, true},
		{"negative_worsens", -100, -125, -25, true},
		{"crosses_zero", -100, 100, 200, true},
		{"large_positive", large, large * 2, 100, true},
		{"large_negative", -large * 2, -large, 50, true},
		{"zero_baseline", 0, 1, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old := content.Gear{Slot: content.SlotHead, Stats: content.Stats{STR: tc.before}}
			candidate := old
			candidate.Stats.STR = tc.after
			got := compareGearAt(candidate, old, true, time.Time{})
			if len(got.Changes) != 1 {
				t.Fatalf("missing changed stat: %+v", got)
			}
			row := got.Changes[0]
			if row.HasPercent != tc.hasPercent || math.IsNaN(row.Percent) || math.IsInf(row.Percent, 0) || math.Abs(row.Percent-tc.percent) > 1e-10 {
				t.Fatalf("percentage = %+v, want %g (available %v)", row, tc.percent, tc.hasPercent)
			}
		})
	}
}

func TestItemQuickwinsTinyXPChangesRetainPrecisionAndClassification(t *testing.T) {
	for _, delta := range []float64{1e-6, 1e-10} {
		for _, direction := range []float64{-1, 1} {
			old := content.Gear{Slot: content.SlotHead, Rarity: content.RarityRare, XPMultiplier: 1.01}
			candidate := old
			candidate.XPMultiplier += direction * delta
			got := compareGearAt(candidate, old, true, time.Time{})
			want := (candidate.XPMultiplier - old.XPMultiplier) * 100
			if got.XPBonusDelta == 0 || math.Abs(float64(got.XPBonusDelta)-want) > math.Abs(want)*1e-6 {
				t.Errorf("tiny XP delta %g lost: got %v, want %.12g percentage points", direction*delta, got.XPBonusDelta, want)
			}
			status := "upgrade"
			if direction < 0 {
				status = "downgrade"
			}
			if got.Status != status || got.IsUpgrade != (direction > 0) {
				t.Errorf("tiny XP delta %g classified %s, want %s", direction*delta, got.Status, status)
			}
		}
	}
}

func TestItemQuickwinsUnknownEffectsAlwaysRequireManualComparison(t *testing.T) {
	unknown := content.ItemEffect("FutureUnrecognizedEffect")
	for _, tc := range []struct {
		name                    string
		occupied, shared, bonus bool
	}{
		{"empty_primary", false, false, false}, {"empty_bonus", false, false, true},
		{"occupied_added_primary", true, false, false}, {"occupied_added_bonus", true, false, true},
		{"occupied_shared_primary", true, true, false}, {"occupied_shared_bonus", true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old := content.Gear{Slot: content.SlotHead, Stats: content.Stats{STR: 10}}
			candidate := old
			candidate.Stats.STR = 20
			if tc.bonus {
				candidate.BonusEffects = []content.ItemEffect{unknown}
				if tc.shared {
					old.BonusEffects = []content.ItemEffect{unknown}
				}
			} else {
				candidate.Special = unknown
				if tc.shared {
					old.Special = unknown
				}
			}
			got := compareGearAt(candidate, old, tc.occupied, time.Time{})
			if got.IsUpgrade || got.Status != "tradeoff" || !strings.Contains(strings.ToLower(strings.Join(got.Reasons, " ")), "effect") {
				t.Fatalf("unknown effect escaped manual comparison: %+v", got)
			}
		})
	}
}
