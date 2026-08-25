package bot

import (
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAppendAbyssFightBreakdown(t *testing.T) {
	t.Parallel()

	base := []string{"summary"}
	if got := appendAbyssFightBreakdown(base, nil); len(got) != 1 {
		t.Fatalf("non-Abyss breakdown added %d lines, want none", len(got)-1)
	}

	got := appendAbyssFightBreakdown(base, &abyssFightTrack{thorns: 125, counters: 80})
	joined := strings.Join(got, "\n")
	for _, want := range []string{"Thorns reflected: 125 damage", "Parry counter-attacks: 80 damage"} {
		if !strings.Contains(joined, want) {
			t.Errorf("breakdown %q does not contain %q", joined, want)
		}
	}
}

func TestAbyssCombatTargetingAndBuildBonuses(t *testing.T) {
	t.Parallel()

	alive := []*content.Mob{{Name: "Summoned Ember Imp", Stats: content.Stats{HP: 10}}, {Name: "Frost Lich", Stats: content.Stats{HP: 10}}}
	if got := petFocusTarget(alive, "ember imp"); got != alive[0] {
		t.Fatalf("pet focus target = %v, want %v", got, alive[0])
	}
	if got := petFocusTarget(alive, "missing"); got != nil {
		t.Fatalf("missing pet focus target = %v, want nil", got)
	}

	equipped := map[content.GearSlot]content.Gear{
		content.SlotMainHand: {Rune: string(content.ElementFire)},
		content.SlotOffHand:  {Rune: string(content.ElementFire)},
		content.SlotChest:    {Rune: string(content.ElementFire)},
	}
	if !runeWardResist(equipped, content.ElementFire) {
		t.Fatal("three matching runes did not activate the elemental ward")
	}
	if runeWardResist(equipped, content.ElementWater) {
		t.Fatal("fire runes activated against water damage")
	}

	crit, damage, lifesteal := abyssFocusMicroBonus("gold")
	if crit != 2 || damage != 1 || lifesteal != 0 {
		t.Fatalf("gold focus bonus = %d, %.2f, %d", crit, damage, lifesteal)
	}
	if canUseHeldManaAbility(true, false, false) {
		t.Fatal("automatic cast bypassed hold-mana on a normal wave")
	}
	if !canUseHeldManaAbility(true, false, true) || !canUseHeldManaAbility(true, true, false) {
		t.Fatal("manual cast or boss cast was blocked by hold-mana")
	}
}

func TestAbyssParryMasteryTracksCountersAndGrantsStealth(t *testing.T) {
	t.Parallel()

	combatant := activeUser{
		u: &UserInCombat{
			UID: "delver", Nickname: "Delver", EscrowLoot: true, CurrentHP: 1_000,
			Stats: content.Stats{HP: 1_000, STR: 20}, Equipped: map[content.GearSlot]content.Gear{},
		},
		effects: []content.ItemEffect{content.EffectParry},
	}
	mob := &content.Mob{Name: "Duelist", Stats: content.Stats{HP: 10_000, STR: 50, SPD: 10}, MaxHP: 10_000, STRMod: 1}
	users, mobs := []activeUser{combatant}, []*content.Mob{mob}
	logs := []string{}
	totalMobDamage, totalUserDamage := 0, 0
	track := &abyssFightTrack{}
	for round := 1; round <= 3; round++ {
		(&Bot{}).mobTurn(users, mobs, content.Zone{}, 1, &logs, &totalMobDamage, &totalUserDamage, round, false, track, fixedCombatRandom{})
	}
	if track.counters <= 0 || users[0].parryCount != 3 || users[0].stealthUntilRound != 4 {
		t.Fatalf("parry mastery = counters %d, parries %d, stealth round %d", track.counters, users[0].parryCount, users[0].stealthUntilRound)
	}
	before := mob.Stats.HP
	(&Bot{}).mobTurn(users, mobs, content.Zone{}, 1, &logs, &totalMobDamage, &totalUserDamage, 4, false, track, fixedCombatRandom{})
	if mob.Stats.HP != before {
		t.Fatalf("stealthed round countered for %d damage", before-mob.Stats.HP)
	}
	if !strings.Contains(strings.Join(logs, "\n"), "Parry mastery") {
		t.Fatalf("parry mastery log missing: %q", logs)
	}
}

func TestAbyssRuneWardReducesIncomingDamage(t *testing.T) {
	t.Parallel()

	equipped := map[content.GearSlot]content.Gear{
		content.SlotMainHand: {Rune: string(content.ElementFire)},
		content.SlotOffHand:  {Rune: string(content.ElementFire)},
		content.SlotRanged:   {Rune: string(content.ElementFire)},
	}
	users := []activeUser{{u: &UserInCombat{
		UID: "warded", Nickname: "Warded", EscrowLoot: true, CurrentHP: 1_000,
		Stats: content.Stats{HP: 1_000}, Equipped: equipped,
	}}}
	mobs := []*content.Mob{{Name: "Flame", Element: content.ElementFire, Stats: content.Stats{HP: 100, STR: 100, SPD: 10}, MaxHP: 100, STRMod: 1}}
	logs := []string{}
	totalMobDamage, totalUserDamage := 0, 0
	(&Bot{}).mobTurn(users, mobs, content.Zone{}, 1, &logs, &totalMobDamage, &totalUserDamage, 1, false, &abyssFightTrack{}, fixedCombatRandom{float: 1, intn: 99})
	if totalMobDamage != 90 || users[0].u.CurrentHP != 910 {
		t.Fatalf("rune-ward damage = %d, HP = %d; want 90, 910", totalMobDamage, users[0].u.CurrentHP)
	}
	if !strings.Contains(strings.Join(logs, "\n"), "three-rune ward resists 10%") {
		t.Fatalf("rune-ward log missing: %q", logs)
	}
}

func TestAbyssCombatHUDContracts(t *testing.T) {
	t.Parallel()

	live, err := webAssets.ReadFile("webassets/abyss_live.html")
	if err != nil {
		t.Fatal(err)
	}
	pixel, err := webAssets.ReadFile("webassets/abyss_pixel.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_live.css")
	if err != nil {
		t.Fatal(err)
	}
	source := string(live) + string(pixel) + string(styles)
	for _, required := range []string{
		"liveUltimateCharge", "renderLiveUltimateCharge", "ab-cooldown-pips",
		"cooldown_max", "dot-active", "ab-dot-stripes", "ENRAGE in 2 rounds", "enrage-imminent",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("Abyss combat HUD is missing %q", required)
		}
	}
}

func TestAppendAbyssFightBreakdownOmitsZeroTotals(t *testing.T) {
	t.Parallel()

	got := appendAbyssFightBreakdown([]string{"summary"}, &abyssFightTrack{})
	if len(got) != 1 {
		t.Fatalf("zero breakdown added %d lines, want none", len(got)-1)
	}
}
