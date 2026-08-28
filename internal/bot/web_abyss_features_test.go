package bot

import (
	"testing"

	"ts3news/internal/content"
)

func TestAbyssRepairDepthMultiplier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		depth int
		want  int64
	}{
		{name: "surface", depth: 0, want: 1},
		{name: "below first pressure tier", depth: 9, want: 1},
		{name: "depth ten", depth: 10, want: 2},
		{name: "depth fifty", depth: 50, want: 26},
		{name: "depth one hundred", depth: 100, want: 101},
		{name: "extreme depth is capped", depth: 10_000, want: 10_001},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := abyssRepairDepthMultiplier(test.depth); got != test.want {
				t.Fatalf("abyssRepairDepthMultiplier(%d) = %d, want %d", test.depth, got, test.want)
			}
		})
	}
}

func TestAbyssRepairRarityMultiplier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		rarity content.Rarity
		want   int64
	}{
		{name: "epic stays base", rarity: content.RarityEpic, want: 1},
		{name: "legendary doubles", rarity: content.RarityLegendary, want: 2},
		{name: "mythic triples", rarity: content.RarityMythic, want: 3},
		{name: "divine quintuple", rarity: content.RarityDivine, want: 5},
		{name: "celestial octuple", rarity: content.RarityCelestial, want: 8},
		{name: "eternal twelvefold", rarity: content.RarityEternal, want: 12},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := abyssRepairRarityMultiplier(test.rarity); got != test.want {
				t.Fatalf("abyssRepairRarityMultiplier(%s) = %d, want %d", test.rarity, got, test.want)
			}
		})
	}
}

func TestAbyssRepairItemCostCombinesDepthAndRarity(t *testing.T) {
	t.Parallel()

	// Two missing points at depth 50: 2 × 200g × 26 depth × 3 Mythic.
	if got, want := abyssRepairItemCost(2, content.RarityMythic, 50), int64(31_200); got != want {
		t.Fatalf("abyssRepairItemCost() = %d, want %d", got, want)
	}
}

func TestAbyssSpecializationCatalogIncludesAddedPaths(t *testing.T) {
	t.Parallel()

	if len(abyssSpecializations) != 13 || len(abyssSpecs) != 13 {
		t.Fatalf("specialization catalog = %d definitions, %d lookup entries; want 13", len(abyssSpecializations), len(abyssSpecs))
	}
	if nodes := abyssSpecializationNodes(); len(nodes) != 27 {
		t.Fatalf("specialization constellation has %d nodes, want 27", len(nodes))
	}
	for _, key := range []string{"berserker", "arcanist", "ranger", "vanguard", "reaver", "chronomancer", "artificer", "oracle", "geomancer", "voidwalker"} {
		if abyssSpecs[key] == "" {
			t.Errorf("added specialization %q is missing", key)
		}
	}
}

func TestApplyAbyssSpecializationPassive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec string
		key  string
		want float64
	}{
		{name: "berserker", spec: "berserker", key: "str_pct", want: 0.08},
		{name: "reaver", spec: "reaver", key: "def_to_lifesteal", want: 0.015},
		{name: "artificer", spec: "artificer", key: "material_yield", want: 0.20},
		{name: "voidwalker", spec: "voidwalker", key: "skill_damage", want: 0.12},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			bonus := content.TreeBonus{}
			applyAbyssSpecializationPassive(test.spec, &bonus)
			if bonus.Pct[test.key] != test.want {
				t.Fatalf("%s %s = %v, want %v", test.spec, test.key, bonus.Pct[test.key], test.want)
			}
		})
	}
}
