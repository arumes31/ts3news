package bot

import (
	"strings"
	"testing"
	"ts3news/internal/content"
)

func TestGearBalancePreviewShowsMarginalChanceAndDuplicateBonus(t *testing.T) {
	current := content.Gear{Slot: content.SlotHead, Special: content.EffectQuick}
	candidate := content.Gear{Slot: content.SlotHead, Special: content.EffectQuick, Stats: content.Stats{CRT: 100, DGE: 50}}
	totals := content.Stats{CRT: 100, DGE: 50}
	notes := gearRatingMarginalNotes(totals, current.Stats, candidate.Stats)
	if len(notes) != 3 || !strings.Contains(notes[0], "33.33% → 40.00%") || !strings.Contains(notes[1], "16.67% → 20.00%") {
		t.Fatalf("incorrect chance preview: %v", notes)
	}
	equipped := map[string]content.Gear{string(content.SlotHead): current, string(content.SlotFeet): {Slot: content.SlotFeet, Special: content.EffectQuick}}
	candidate.Special = content.EffectNone
	notes = gearPassiveMarginalNotes(candidate, equipped)
	if len(notes) != 1 || !strings.Contains(notes[0], "15.00% → 10.00%") {
		t.Fatalf("incorrect duplicate preview: %v", notes)
	}
}

func TestGearPreviewUsesEffectiveAffixBudget(t *testing.T) {
	g := content.Gear{Slot: content.SlotHead, Rarity: content.RarityMythic, Special: content.EffectQuick, BonusEffects: []content.ItemEffect{content.EffectLucky, content.EffectBulwark, content.EffectVampiric}}
	if len(gearSpecialViews(g)) != 3 {
		t.Fatal("UI displays an inactive affix")
	}
}

func TestCombatRollsUseFractionalRatingChance(t *testing.T) {
	for _, roll := range []float64{.124, .125, .249, .25} {
		live := &abyssLiveCombat{round: 2}
		user := &UserInCombat{UID: "owner", Nickname: "Delver", shadow: true, EscrowLoot: true, live: live, CurrentHP: 100, Stats: content.Stats{HP: 100, STR: 100, SPD: 10, CRT: 50, DGE: 25}, STRMod: 1, DEFMod: 1, SPDMod: 1}
		users := []activeUser{{u: user, CurrentMana: 100, MaxMana: 100, skillCooldowns: map[string]int{}}}
		mob := &content.Mob{Name: "Target", Stats: content.Stats{HP: 10000, STR: 10, SPD: 10}, MaxHP: 10000, STRMod: 1, DEFMod: 1, SPDMod: 1}
		mobs := []*content.Mob{mob}
		logs := []string{}
		loot := []LootResult{}
		dealt, taken := 0, 0
		(&Bot{}).userTurn(users, &mobs, content.Zone{}, 1, 1, &logs, &dealt, &taken, 1, 1, nil, &loot, 2, nil, map[string]abyssLiveAction{"owner": {Kind: "attack", TargetID: "enemy:0", Round: 2}}, false, fixedCombatRandom{float: roll, intn: 99})
		if len(live.presentationEvents) != 1 || live.presentationEvents[0].Targets[0].Critical != (roll < .25) {
			t.Fatalf("CRT50 roll%v: %v", roll, live.presentationEvents)
		}
		live.presentationEvents = nil
		(&Bot{}).mobTurn(users, mobs, content.Zone{}, 1, &logs, &taken, &dealt, 2, false, nil, fixedCombatRandom{float: roll, intn: 99})
		if len(live.presentationEvents) != 1 || live.presentationEvents[0].Targets[0].Dodged != (roll < .125) {
			t.Fatalf("DGE25 roll%v: %v", roll, live.presentationEvents)
		}
	}
}
