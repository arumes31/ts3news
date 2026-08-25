package bot

import (
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssPetAbilitiesExposeFormationCooldowns(t *testing.T) {
	pounce, ok := abyssPetAbilityForSlot(1)
	if !ok || pounce.Name != "Pounce" || pounce.Kind != "attack" || pounce.Cooldown != 3 || pounce.PowerScale != 1.5 {
		t.Fatalf("slot one ability = %#v, %v", pounce, ok)
	}
	heal, ok := abyssPetAbilityForSlot(2)
	if !ok || heal.Name != "Healing Spell" || heal.Kind != "heal" || heal.Cooldown != 2 || heal.PowerScale != 0.15 {
		t.Fatalf("slot two ability = %#v, %v", heal, ok)
	}
	if _, ok := abyssPetAbilityForSlot(0); ok {
		t.Fatal("reserve companion received an active ability")
	}
}

func TestAbyssPetAbilityCooldownsTickIndependentlyAndLogReady(t *testing.T) {
	user := &activeUser{
		u:            &UserInCombat{Pets: []*content.Mob{{Name: "Fang"}, {Name: "Mender"}}},
		petCooldowns: map[int]int{0: 2, 1: 1},
	}
	first := tickAbyssPetAbilityCooldowns(user)
	if len(first) != 1 || !strings.Contains(first[0], "Mender's Healing Spell is ready") {
		t.Fatalf("first cooldown tick logs = %v", first)
	}
	if user.petCooldowns[0] != 1 {
		t.Fatalf("Pounce cooldown = %d, want 1", user.petCooldowns[0])
	}
	second := tickAbyssPetAbilityCooldowns(user)
	if len(second) != 1 || !strings.Contains(second[0], "Fang's Pounce is ready") {
		t.Fatalf("second cooldown tick logs = %v", second)
	}
	if len(user.petCooldowns) != 0 {
		t.Fatalf("expired cooldowns remain: %v", user.petCooldowns)
	}
}

func TestAbyssPetAbilityUIShowsCooldowns(t *testing.T) {
	if got := abyssPetAbilityLabel(1); got != "Pounce · 3-round cooldown" {
		t.Fatalf("Pounce label = %q", got)
	}
	if got := abyssPetAbilityLabel(2); got != "Healing Spell · 2-round cooldown" {
		t.Fatalf("Healing Spell label = %q", got)
	}
	page, err := webAssets.ReadFile("webassets/abyss_social.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"{{.Ability}}", "Ability cooldowns tick once after every combat round"} {
		if !strings.Contains(string(page), token) {
			t.Errorf("companion ability UI is missing %q", token)
		}
	}
}

func TestSetAbyssPetAbilityCooldownInitializesStorage(t *testing.T) {
	user := &activeUser{}
	setAbyssPetAbilityCooldown(user, 0, 3)
	if user.petCooldowns[0] != 3 {
		t.Fatalf("stored cooldowns = %v", user.petCooldowns)
	}
}
