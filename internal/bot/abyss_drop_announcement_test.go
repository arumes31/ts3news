package bot

import (
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssHighRarityDropFanfareAnnouncesMythicOrHigherAndEscapesBBCode(t *testing.T) {
	t.Parallel()

	for _, rarity := range []content.Rarity{content.RarityCommon, content.RarityLegendary} {
		if _, _, ok := abyssHighRarityDropFanfare("Delver", "Rare Blade", rarity); ok {
			t.Fatalf("%s gear triggered Mythic+ fanfare", rarity)
		}
	}
	for _, rarity := range []content.Rarity{content.RarityMythic, content.RarityDivine, content.RarityCelestial, content.RarityEternal} {
		nickname, message, ok := abyssHighRarityDropFanfare("[url=x]Delver[/url]", "[b]Relic[/b]", rarity)
		if !ok || !strings.Contains(nickname, rarity.String()) || !strings.Contains(message, strings.ToUpper(rarity.String())) {
			t.Fatalf("%s fanfare = %q, %q, %v", rarity, nickname, message, ok)
		}
		if strings.Contains(message, "[url=") || strings.Contains(message, "[b]") {
			t.Fatalf("%s fanfare retained BBCode: %q", rarity, message)
		}
	}
}

func TestAbyssHighRarityEscrowDropClassifiesMythicOrHigherGear(t *testing.T) {
	t.Parallel()

	mythic := content.Gear{Name: "Void Crown", Rarity: content.RarityMythic}
	legendary := content.Gear{Name: "Sun Blade", Rarity: content.RarityLegendary}
	unidentified := content.Gear{Name: "Secret Crown", Slot: content.SlotHead, Rarity: content.RarityCelestial, Unidentified: true}
	unidentifiedWithoutSlot := content.Gear{Name: "Secret Relic", Rarity: content.RarityEternal, Unidentified: true}
	tests := []struct {
		name     string
		grant    abyssLootGrant
		wantName string
		want     bool
	}{
		{name: "Mythic gear", grant: abyssLootGrant{Type: "gear", Gear: &mythic}, wantName: mythic.Name, want: true},
		{name: "unidentified Celestial gear", grant: abyssLootGrant{Type: "gear", Gear: &unidentified}, wantName: "Unidentified Head", want: true},
		{name: "unidentified gear without slot", grant: abyssLootGrant{Type: "gear", Gear: &unidentifiedWithoutSlot}, wantName: "Unidentified gear", want: true},
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
			if ok && (name != test.wantName || rarity != test.grant.Gear.Rarity) {
				t.Fatalf("classified payload = %q, %s", name, rarity)
			}
			if strings.Contains(name, "Secret") {
				t.Fatalf("classification leaked unidentified item name: %q", name)
			}
		})
	}
}
