package bot

import (
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssHighRarityDropFanfareRemainsEternalOnlyAndEscapesBBCode(t *testing.T) {
	t.Parallel()

	for _, rarity := range []content.Rarity{content.RarityLegendary, content.RarityMythic, content.RarityCelestial} {
		if _, _, ok := abyssHighRarityDropFanfare("Delver", "Rare Blade", rarity); ok {
			t.Fatalf("%s gear triggered Eternal-only fanfare", rarity)
		}
	}
	for _, rarity := range []content.Rarity{content.RarityEternal} {
		nickname, message, ok := abyssHighRarityDropFanfare("[url=x]Delver[/url]", "[b]Relic[/b]", rarity)
		if !ok || !strings.Contains(nickname, rarity.String()) || !strings.Contains(message, strings.ToUpper(rarity.String())) {
			t.Fatalf("%s fanfare = %q, %q, %v", rarity, nickname, message, ok)
		}
		if strings.Contains(message, "[url=") || strings.Contains(message, "[b]") {
			t.Fatalf("%s fanfare retained BBCode: %q", rarity, message)
		}
	}
}

func TestAbyssHighRarityEscrowDropClassifiesOnlyEternalGear(t *testing.T) {
	t.Parallel()

	mythic := content.Gear{Name: "Void Crown", Rarity: content.RarityMythic}
	legendary := content.Gear{Name: "Sun Blade", Rarity: content.RarityLegendary}
	eternal := content.Gear{Name: "First Light", Rarity: content.RarityEternal}
	tests := []struct {
		name  string
		grant abyssLootGrant
		want  bool
	}{
		{name: "Eternal gear", grant: abyssLootGrant{Type: "gear", Gear: &eternal}, want: true},
		{name: "Mythic gear", grant: abyssLootGrant{Type: "gear", Gear: &mythic}},
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
			if ok && (name != eternal.Name || rarity != content.RarityEternal) {
				t.Fatalf("classified payload = %q, %s", name, rarity)
			}
		})
	}
}
