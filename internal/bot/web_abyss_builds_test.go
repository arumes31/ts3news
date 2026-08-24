package bot

import (
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestApplyAbyssRunBuildUsesOneKitAndMutation(t *testing.T) {
	t.Parallel()

	baseSkill := content.Skill{ID: "fire", Name: "Fire Bolt", Power: 2, IgnoreDef: 0.1}
	u := UserInCombat{
		Stats:  content.Stats{HP: 100, STR: 50, DEF: 40, INT: 100, MNA: 20},
		Skills: []content.Skill{baseSkill},
	}
	flags := map[string]int64{
		abyssRunFlagBuildKit:      abyssBuildKits["arcanist"],
		abyssRunFlagSkillMutation: abyssSkillMutations["piercing"],
	}
	applyAbyssRunBuild(&u, flags, map[string]int{"fire": 50})
	if u.Stats.INT != 110 || u.Stats.MNA != 50 || u.Stats.STR != 50 {
		t.Fatalf("arcanist kit applied unexpected stats: %#v", u.Stats)
	}
	if got := u.Skills[0]; got.IgnoreDef != 0.25 || got.Power != 2.2 || got.HealPercent != 0 {
		t.Fatalf("piercing mutation/mastery overlap incorrectly: %#v", got)
	}
}

func TestAbyssElementReactionsAndComboTags(t *testing.T) {
	t.Parallel()

	if got := abyssElementReaction(content.ElementFire, content.ElementWater); got != "Steam Burst" {
		t.Fatalf("fire/water reaction = %q", got)
	}
	if got := abyssElementReaction(content.ElementFire, content.ElementFire); got != "" {
		t.Fatalf("same element unexpectedly reacted: %q", got)
	}
	tags := abyssSkillComboTags(content.Skill{
		Name: "Storm Sunder", IgnoreDef: 0.2, StunChance: 0.1,
	})
	joined := strings.Join(tags, ",")
	for _, want := range []string{"air", "control", "setup", "armor-break", "finisher"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("combo tags %q missing %q", joined, want)
		}
	}
}

func TestAbyssPartyBuildSynergyRequiresDistinctIdentities(t *testing.T) {
	t.Parallel()

	users := []UserInCombat{
		{UID: "a", Stats: content.Stats{HP: 100, STR: 100}},
		{UID: "b", Stats: content.Stats{HP: 100, INT: 100}},
	}
	flags := map[string]map[string]int64{
		"a": {abyssRunFlagBuildKit: abyssBuildKits["vanguard"]},
		"b": {abyssRunFlagBuildKit: abyssBuildKits["arcanist"]},
	}
	if _, active := applyAbyssPartyBuildSynergy(users, flags); !active {
		t.Fatal("distinct kits did not activate party synergy")
	}
	if users[0].Stats.HP != 105 || users[1].Stats.INT != 105 {
		t.Fatalf("party synergy did not scale both builds: %#v", users)
	}

	sameFlags := map[string]map[string]int64{
		"a": {abyssRunFlagBuildKit: abyssBuildKits["vanguard"]},
		"b": {abyssRunFlagBuildKit: abyssBuildKits["vanguard"]},
	}
	if _, active := applyAbyssPartyBuildSynergy(users, sameFlags); active {
		t.Fatal("identical kits activated cross-class synergy")
	}
}

func TestAbyssBuildSummaryIncludesMutationRelicAndSet(t *testing.T) {
	t.Parallel()

	u := UserInCombat{Equipped: map[content.GearSlot]content.Gear{
		content.SlotRelic:    {Name: "Heart of Test", Slot: content.SlotRelic},
		content.SlotMainHand: {Name: "Fang", Slot: content.SlotMainHand, SetID: "predator"},
		content.SlotHead:     {Name: "Hood", Slot: content.SlotHead, SetID: "predator"},
	}}
	flags := map[string]int64{
		abyssRunFlagBuildKit:      abyssBuildKits["vanguard"],
		abyssRunFlagSkillMutation: abyssSkillMutations["empowered"],
	}
	summary := abyssBuildSummary(u, flags)
	for _, want := range []string{"Vanguard kit", "empowered skills", "Heart of Test active", "predator 2-piece synergy"} {
		if !strings.Contains(summary, want) {
			t.Fatalf("build summary %q missing %q", summary, want)
		}
	}
}
