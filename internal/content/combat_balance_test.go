package content

import (
	"math"
	"testing"
)

func TestCombatRatingChanceRemainsUsefulAboveLegacyCap(t *testing.T) {
	for _, tc := range []struct {
		name   string
		chance func(int) float64
		half   int
		cap    float64
	}{{"crit", CriticalChance, 50, 50}, {"dodge", DodgeChance, 25, 25}} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.chance(-1) != 0 || tc.chance(0) != 0 || tc.chance(tc.half) != tc.cap/2 {
				t.Fatal("invalid rating anchors")
			}
			previous := 0.0
			for _, rating := range []int{1, 5, 25, 50, 100, 1000, 10000, 1000000} {
				value := tc.chance(rating)
				if value <= previous || value >= tc.cap {
					t.Fatalf("rating %d gives %v after %v", rating, value, previous)
				}
				previous = value
			}
		})
	}
}

func TestPassiveBonusesDiminishAndCap(t *testing.T) {
	for _, tc := range []struct {
		effect     ItemEffect
		first, cap float64
	}{{EffectQuick, .1, .4}, {EffectVampiric, .05, .3}, {EffectBerserk, .2, .4}, {EffectFragile, .3, .4}, {EffectExecutioner, .25, .4}, {EffectThorns, .1, .4}, {EffectTreasureHunter, .05, .4}} {
		effects := []ItemEffect{tc.effect}
		first := PassiveEffectBonus(effects, tc.effect)
		effects = append(effects, tc.effect)
		second := PassiveEffectBonus(effects, tc.effect)
		if math.Abs(first-tc.first) > 1e-9 || second <= first || second-first >= first {
			t.Fatalf("bad diminishing returns: %v %v", first, second)
		}
		for range 10000 {
			effects = append(effects, tc.effect)
		}
		if PassiveEffectBonus(effects, tc.effect) > tc.cap {
			t.Fatal("exceeds cap")
		}
	}
}

func TestAscensionBonusAffixBudget(t *testing.T) {
	g := Gear{Rarity: RarityLegendary, Special: EffectVampiric}
	for _, rarity := range []Rarity{RarityMythic, RarityDivine, RarityCelestial, RarityEternal} {
		g.Rarity = rarity
		AddBonusEffects(&g, 99, func(int) int { return 0 })
		if len(g.BonusEffects) != BonusEffectBudget(rarity) {
			t.Fatalf("%v has %d affixes", rarity, len(g.BonusEffects))
		}
		for _, effect := range g.BonusEffects {
			if effect == g.Special {
				t.Fatal("native special duplicated")
			}
		}
	}
	if g.Special != EffectVampiric {
		t.Fatal("native special lost")
	}
	legacy := Gear{Rarity: RarityEternal, Special: EffectVampiric, BonusEffects: append(g.BonusEffects, EffectQuick, EffectQuick, EffectNone, EffectVampiric)}
	if len(legacy.Effects()) != 5 {
		t.Fatalf("legacy effects not bounded/deduplicated: %v", legacy.Effects())
	}
}

func TestAffixCapacityIgnoresNativeAndDuplicateStoredEntries(t *testing.T) {
	g := Gear{Rarity: RarityMythic, Special: EffectVampiric, BonusEffects: []ItemEffect{EffectVampiric, EffectNone, EffectQuick, EffectQuick}}
	AddBonusEffects(&g, 1, func(int) int { return 0 })
	if len(g.AddedEffects()) != 2 || len(g.Effects()) != 3 {
		t.Fatalf("phantom entries consumed capacity: %v", g)
	}
	legacy := Gear{Rarity: RarityMythic, BonusEffects: []ItemEffect{EffectQuick, EffectLucky, EffectBulwark, EffectFocused}}
	AddBonusEffects(&legacy, 1, func(int) int { return 0 })
	if len(legacy.BonusEffects) != 4 || len(legacy.Effects()) != 2 {
		t.Fatal("distinct legacy roll discarded or activated above budget")
	}
}

func TestPassiveStatsApplyAggregateOnceWithoutCompounding(t *testing.T) {
	stats := Stats{SPD: 100, DEF: 100, LCK: 100, CRT: 100}
	effects := []ItemEffect{EffectQuick, EffectQuick, EffectBulwark, EffectLucky, EffectFocused}
	actual := ApplyPassiveStats(stats, effects)
	if actual.SPD != 115 || actual.DEF != 110 || actual.LCK != 110 || actual.CRT != 110 {
		t.Fatalf("aggregate rounding or compounding changed stats: %+v", actual)
	}
	for range 10000 {
		effects = append(effects, EffectQuick)
	}
	if actual = ApplyPassiveStats(stats, effects); actual.SPD != 140 {
		t.Fatalf("stat cap = %d, want140", actual.SPD)
	}
}
