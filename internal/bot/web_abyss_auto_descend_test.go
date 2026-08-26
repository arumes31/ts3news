package bot

import (
	"testing"

	"ts3news/internal/content"
)

func TestAbyssAutoDescendRulesValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		rules abyssAutoDescendRules
		want  bool
	}{
		{name: "disabled", rules: abyssAutoDescendRules{}, want: true},
		{name: "queue end", rules: abyssAutoDescendRules{HPBelowPct: 50, TargetDepth: 17, StopOnLegendary: true}, want: true},
		{name: "invalid HP", rules: abyssAutoDescendRules{HPBelowPct: 100}},
		{name: "past depth", rules: abyssAutoDescendRules{TargetDepth: 12}},
		{name: "beyond queue", rules: abyssAutoDescendRules{TargetDepth: 18}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rules.validate(12, 5)
			if (err == nil) != tt.want {
				t.Fatalf("validate() error = %v, want valid %t", err, tt.want)
			}
		})
	}
}

func TestAbyssAutoDescendStopReason(t *testing.T) {
	t.Parallel()
	rules := abyssAutoDescendRules{HPBelowPct: 50, TargetDepth: 17, StopOnLegendary: true}
	if got := rules.stopReason(13, 500, 1000, false); got != "" {
		t.Fatalf("equal HP threshold stopped with %q", got)
	}
	if got := rules.stopReason(13, 499, 1000, false); got != abyssAutoStopHP {
		t.Fatalf("low HP reason = %q", got)
	}
	if got := rules.stopReason(13, 900, 1000, true); got != abyssAutoStopLegendary {
		t.Fatalf("legendary reason = %q", got)
	}
	if got := rules.stopReason(17, 900, 1000, false); got != abyssAutoStopDepth {
		t.Fatalf("depth reason = %q", got)
	}
}

func TestAbyssLootGrantLegendaryClassification(t *testing.T) {
	t.Parallel()
	legendaryGear := content.Gear{Rarity: content.RarityLegendary}
	rareGear := content.Gear{Rarity: content.RarityRare}
	legendarySkill := content.Skill{Rarity: content.RarityLegendary}
	legendaryEnchantment := content.Enchantment{Rarity: content.RarityMythic}
	tests := []struct {
		name  string
		grant abyssLootGrant
		want  bool
	}{
		{name: "legendary gear", grant: abyssLootGrant{Gear: &legendaryGear}, want: true},
		{name: "rare gear", grant: abyssLootGrant{Gear: &rareGear}},
		{name: "legendary skill", grant: abyssLootGrant{Skill: &legendarySkill}, want: true},
		{name: "mythic enchantment", grant: abyssLootGrant{Ench: &legendaryEnchantment}, want: true},
		{name: "legendary unique", grant: abyssLootGrant{UniqName: "Relic", UniqRar: content.RarityLegendary}, want: true},
		{name: "materials", grant: abyssLootGrant{Type: "mat", MatID: "core"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := abyssLootGrantIsLegendary(tt.grant); got != tt.want {
				t.Fatalf("abyssLootGrantIsLegendary() = %t, want %t", got, tt.want)
			}
		})
	}
}
