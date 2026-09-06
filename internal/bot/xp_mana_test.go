package bot

import (
	"testing"

	"ts3news/internal/content"
)

func TestCombatManaRegen(t *testing.T) {
	for _, test := range []struct {
		name string
		mana int
		want int
	}{
		{name: "base", want: 10},
		{name: "high mana build", mana: 1000, want: 60},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := combatManaRegen(&UserInCombat{Stats: content.Stats{MNA: test.mana}}); got != test.want {
				t.Fatalf("regen = %d, want %d", got, test.want)
			}
		})
	}
}

func TestCombatSkillManaCost(t *testing.T) {
	for _, test := range []struct {
		name    string
		base    int
		insight int
		robes   bool
		want    int
	}{
		{name: "default", want: 20},
		{name: "authored cost", base: 35, want: 35},
		{name: "robes", base: 35, robes: true, want: 30},
		{name: "insight", base: 35, insight: 1, want: 33},
		{name: "minimum cost", base: 5, robes: true, want: 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			user := &activeUser{u: &UserInCombat{Equipped: map[content.GearSlot]content.Gear{}}}
			if test.robes {
				user.u.Equipped[content.SlotChest] = content.Gear{ID: "ABYSS_ARCHMAGE_ROBES"}
			}
			if got := combatSkillManaCost(user, test.base, test.insight); got != test.want {
				t.Fatalf("cost = %d, want %d", got, test.want)
			}
		})
	}
}
