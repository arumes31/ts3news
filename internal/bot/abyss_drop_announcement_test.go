package bot

import (
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssHighRarityDropFanfareStartsAtMythicAndEscapesBBCode(t *testing.T) {
	t.Parallel()

	if _, _, ok := abyssHighRarityDropFanfare("Delver", "Legendary Blade", content.RarityLegendary); ok {
		t.Fatal("Legendary gear triggered Mythic+ fanfare")
	}
	for _, rarity := range []content.Rarity{content.RarityMythic, content.RarityCelestial, content.RarityEternal} {
		nickname, message, ok := abyssHighRarityDropFanfare("[url=x]Delver[/url]", "[b]Relic[/b]", rarity)
		if !ok || !strings.Contains(nickname, rarity.String()) || !strings.Contains(message, strings.ToUpper(rarity.String())) {
			t.Fatalf("%s fanfare = %q, %q, %v", rarity, nickname, message, ok)
		}
		if strings.Contains(message, "[url=") || strings.Contains(message, "[b]") {
			t.Fatalf("%s fanfare retained BBCode: %q", rarity, message)
		}
	}
}

func TestAbyssHighRarityEscrowDropClassifiesOnlyMythicGear(t *testing.T) {
	t.Parallel()

	mythic := content.Gear{Name: "Void Crown", Rarity: content.RarityMythic}
	legendary := content.Gear{Name: "Sun Blade", Rarity: content.RarityLegendary}
	tests := []struct {
		name  string
		grant abyssLootGrant
		want  bool
	}{
		{name: "Mythic gear", grant: abyssLootGrant{Type: "gear", Gear: &mythic}, want: true},
		{name: "Legendary gear", grant: abyssLootGrant{Type: "gear", Gear: &legendary}},
		{name: "gear without payload", grant: abyssLootGrant{Type: "gear"}},
		{name: "non-gear payload", grant: abyssLootGrant{Type: "cons", Gear: &mythic}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name, rarity, ok := abyssHighRarityEscrowDrop(test.grant)
			if ok != test.want {
				t.Fatalf("classification = %q, %s, %v; want %v", name, rarity, ok, test.want)
			}
			if ok && (name != mythic.Name || rarity != content.RarityMythic) {
				t.Fatalf("classified payload = %q, %s", name, rarity)
			}
		})
	}
}
