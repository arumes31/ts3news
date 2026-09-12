package bot

import (
	"math"
	"testing"

	"ts3news/internal/content"
)

func TestForgePreviewTargetOnlyAccuracy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		operation string
		before    content.Stats
		after     content.Stats
	}{
		{
			name: "reinforce changes only DEF", operation: "reinforce",
			before: content.Stats{HP: 200, STR: 100, DEF: 199, MNA: 100, CHA: 30},
			after:  content.Stats{HP: 200, STR: 100, DEF: 202, MNA: 100, CHA: 30},
		},
		{
			name: "reinforce minimum increment on zero", operation: "reinforce",
			before: content.Stats{STR: 100}, after: content.Stats{STR: 100, DEF: 1},
		},
		{
			name: "reinforce minimum increment on negative DEF", operation: "reinforce",
			before: content.Stats{DEF: -100}, after: content.Stats{DEF: -99},
		},
		{
			name: "sharpen changes only STR", operation: "sharpen",
			before: content.Stats{HP: 200, STR: 199, DEF: 100, MNA: 100, STN: 30},
			after:  content.Stats{HP: 200, STR: 202, DEF: 100, MNA: 100, STN: 30},
		},
		{
			name: "sharpen minimum increment on small STR", operation: "sharpen",
			before: content.Stats{STR: 1}, after: content.Stats{STR: 2},
		},
		{
			name: "sharpen minimum increment on negative STR", operation: "sharpen",
			before: content.Stats{STR: -100}, after: content.Stats{STR: -99},
		},
		{
			name: "prismatic HP wins but other stats remain", operation: "prismatic_rune",
			before: content.Stats{HP: 199, STR: 100, DEF: 100, MNA: 100, CHA: 1000},
			after:  content.Stats{HP: 208, STR: 100, DEF: 100, MNA: 100, CHA: 1000},
		},
		{
			name: "prismatic STR wins a tie", operation: "prismatic_rune",
			before: content.Stats{HP: 100, STR: 100, DEF: 100},
			after:  content.Stats{HP: 100, STR: 105, DEF: 100},
		},
		{
			name: "prismatic HP wins ties after STR", operation: "prismatic_rune",
			before: content.Stats{HP: 100, STR: 50, DEF: 100},
			after:  content.Stats{HP: 105, STR: 50, DEF: 100},
		},
		{
			name: "prismatic includes mana", operation: "prismatic_rune",
			before: content.Stats{STR: 100, MNA: 199},
			after:  content.Stats{STR: 100, MNA: 208},
		},
		{
			name: "prismatic minimum increment on zero", operation: "prismatic_rune",
			before: content.Stats{}, after: content.Stats{STR: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gear := content.Gear{Rarity: content.RarityCommon, Stats: tt.before}
			outcome := forgeQuoteOutcome(tt.operation, &gear, 1)
			assertForgePreviewCertainStats(t, outcome, tt.after)
			if gear.Stats != tt.before {
				t.Errorf("preview mutated the source item: got %+v, want %+v", gear.Stats, tt.before)
			}
		})
	}
}

func TestForgePreviewGuaranteedScalingAccuracy(t *testing.T) {
	t.Parallel()
	before := content.Stats{HP: 199, STR: 99, DEF: -99, SPD: 1, LCK: 1, INT: 1, STA: 1, CRT: 1, DGE: 1, MNA: 99, CHA: 77, STN: -77, SHN: 88, HGR: -88}
	tests := []struct {
		name  string
		after content.Stats
	}{
		{
			name: "masterwork",
			after: content.Stats{HP: 204, STR: 101, DEF: -101, SPD: 1, LCK: 1, INT: 1, STA: 1, CRT: 1, DGE: 1, MNA: 101,
				CHA: 77, STN: -77, SHN: 88, HGR: -88},
		},
		{
			name: "attune",
			after: content.Stats{HP: 208, STR: 103, DEF: -103, SPD: 1, LCK: 1, INT: 1, STA: 1, CRT: 1, DGE: 1, MNA: 103,
				CHA: 77, STN: -77, SHN: 88, HGR: -88},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gear := content.Gear{Rarity: content.RarityRare, Stats: before}
			outcome := forgeQuoteOutcome(tt.name, &gear, 1)
			assertForgePreviewCertainStats(t, outcome, tt.after)
			after := content.Gear{Rarity: gear.Rarity, Stats: tt.after}
			assertForgePreviewCR(t, outcome, after.CombatRating(), after.CombatRating(), after.CombatRating())
		})
	}
}

func TestForgePreviewTemperWeightsActualRoundedOutcomes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                             string
		strength                         int
		chance                           float64
		minimum, expected, maximum       int
		minimumCR, expectedCR, maximumCR float64
	}{
		// 99*1.02 truncates to 100. The 75% expectation is 99.75,
		// represented as 99 in integer Stats; scaling 99 by 1.015 gives 100.
		{name: "positive", strength: 99, chance: .75, minimum: 99, expected: 99, maximum: 100,
			minimumCR: 118.8, expectedCR: 119.7, maximumCR: 120},
		{name: "negative", strength: -99, chance: .75, minimum: -100, expected: -99, maximum: -99,
			minimumCR: -120, expectedCR: -119.7, maximumCR: -118.8},
		{name: "small stat cannot grow", strength: 1, chance: .75, minimum: 1, expected: 1, maximum: 1,
			minimumCR: 1.2, expectedCR: 1.2, maximumCR: 1.2},
		{name: "certain success", strength: 99, chance: 1, minimum: 100, expected: 100, maximum: 100,
			minimumCR: 120, expectedCR: 120, maximumCR: 120},
		{name: "certain failure", strength: 99, chance: 0, minimum: 99, expected: 99, maximum: 99,
			minimumCR: 118.8, expectedCR: 118.8, maximumCR: 118.8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gear := content.Gear{Rarity: content.RarityCommon, Stats: content.Stats{STR: tt.strength, CHA: 77}}
			outcome := forgeQuoteOutcome("temper", &gear, tt.chance)
			for _, result := range []struct {
				name string
				got  content.Stats
				want int
			}{
				{"minimum", outcome.MinimumStats, tt.minimum},
				{"expected", outcome.ExpectedStats, tt.expected},
				{"maximum", outcome.MaximumStats, tt.maximum},
			} {
				want := content.Stats{STR: result.want, CHA: 77}
				if result.got != want {
					t.Errorf("%s stats = %+v, want %+v", result.name, result.got, want)
				}
			}
			assertForgePreviewCR(t, outcome, tt.minimumCR, tt.expectedCR, tt.maximumCR)
		})
	}
}

func TestForgePreviewAscensionUsesTargetRarity(t *testing.T) {
	t.Parallel()
	gear := content.Gear{Rarity: content.RarityCommon, Stats: content.Stats{STR: 100, MNA: -99, CHA: 77}}
	outcome := forgeQuoteOutcome("upgrade_gear", &gear, 1)
	assertForgePreviewCertainStats(t, outcome, content.Stats{STR: 130, MNA: -128, CHA: 77})
	// 130 STR * 1.2 CR/STR * 1.25 Uncommon rarity = 195 CR.
	assertForgePreviewCR(t, outcome, 195, 195, 195)
	if gear.Rarity != content.RarityCommon || gear.Stats.STR != 100 {
		t.Errorf("ascension preview mutated its source: %+v", gear)
	}
}

func assertForgePreviewCertainStats(t *testing.T, outcome abyssForgeOutcome, want content.Stats) {
	t.Helper()
	for _, result := range []struct {
		name string
		got  content.Stats
	}{
		{"minimum", outcome.MinimumStats},
		{"expected", outcome.ExpectedStats},
		{"maximum", outcome.MaximumStats},
	} {
		if result.got != want {
			t.Errorf("guaranteed %s stats = %+v, want %+v", result.name, result.got, want)
		}
	}
}

func assertForgePreviewCR(t *testing.T, outcome abyssForgeOutcome, minimum, expected, maximum float64) {
	t.Helper()
	for _, result := range []struct {
		name      string
		got, want float64
	}{
		{"minimum", outcome.MinimumCR, minimum},
		{"expected", outcome.ExpectedCR, expected},
		{"maximum", outcome.MaximumCR, maximum},
	} {
		if math.Abs(result.got-result.want) > 1e-9 {
			t.Errorf("%s CR = %v, want %v", result.name, result.got, result.want)
		}
	}
}
