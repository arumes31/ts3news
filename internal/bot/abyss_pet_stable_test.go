package bot

import (
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssPetStableRules(t *testing.T) {
	if !abyssPetNameValid("Moss-Fang 2") {
		t.Fatal("safe companion name was rejected")
	}
	for _, name := range []string{"", strings.Repeat("a", 25), "Moss<script>", "Sh it"} {
		if abyssPetNameValid(name) {
			t.Errorf("unsafe companion name %q was accepted", name)
		}
	}
	if got := abyssPetCaptureLimitWithBonus(3, 2); got != 5 {
		t.Fatalf("talent capture capacity = %d, want 5", got)
	}
	if got := abyssPetCaptureLimitWithBonus(9, 9); got != abyssPetMaxCap {
		t.Fatalf("capture capacity exceeded cap: %d", got)
	}
	if got := abyssPetBetrayalChance(100, 0); got != 0 {
		t.Fatalf("fully loyal companion betrayal chance = %f", got)
	}
	if abyssPetBetrayalChance(20, 0) <= abyssPetBetrayalChance(80, 0) {
		t.Fatal("loyalty did not lower betrayal risk")
	}
}

func TestAbyssPetClassesAndSupportAbility(t *testing.T) {
	tests := []struct {
		mobType content.MobType
		class   string
	}{{content.MobCommon, "support"}, {content.MobElite, "damage"}, {content.MobBoss, "tank"}}
	for _, test := range tests {
		if got := abyssPetClass(test.mobType); got != test.class {
			t.Errorf("class for %s = %q, want %q", test.mobType, got, test.class)
		}
	}
	ability, ok := abyssPetAbilityForClass(1, "support")
	if !ok || ability.Kind != "heal" || ability.Name != "Mending Cry" {
		t.Fatalf("support ability = %#v, %v", ability, ok)
	}
}

func TestAbyssPetProfileRoundTrip(t *testing.T) {
	want := abyssPetProfile{XP: 75, Favorite: true, Shiny: true, BarkStyle: "bold"}
	encoded, err := encodeAbyssPetProfile(want)
	if err != nil {
		t.Fatal(err)
	}
	got := decodeAbyssPetProfile(encoded)
	if got.XP != want.XP || !got.Favorite || !got.Shiny || got.BarkStyle != "bold" {
		t.Fatalf("profile round trip = %#v", got)
	}
}
