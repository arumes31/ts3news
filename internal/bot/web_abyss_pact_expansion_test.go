package bot

import (
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssChallengePactCatalog(t *testing.T) {
	t.Parallel()

	want := map[string]float64{
		"abstinence":   0.15,
		"pauper":       0.30,
		"anemic":       0.25,
		"cursed_horde": 0.20,
		"deep_drums":   0.35,
		"uninsured":    0.15,
		"blind":        0.10,
		"brittle":      0.10,
		"famine":       0.20,
	}
	for key, reward := range want {
		pact, ok := abyssPactByKey(key)
		if !ok {
			t.Errorf("pact %q is missing", key)
			continue
		}
		if pact.Reward != reward {
			t.Errorf("pact %q reward = %v, want %v", key, pact.Reward, reward)
		}
	}
	canonical := abyssValidatePacts([]string{"famine", "abstinence", "famine", "unknown"})
	if fields := strings.Fields(canonical); len(fields) != 2 || fields[0] != "abstinence" || fields[1] != "famine" {
		t.Fatalf("canonical pacts = %q", canonical)
	}
}

func TestAbyssPauperRejectsHighRarityGear(t *testing.T) {
	t.Parallel()

	gear := map[content.GearSlot]content.Gear{
		content.SlotMainHand: {Name: "Rare blade", Rarity: content.RarityRare},
	}
	if got := abyssPactEquipmentError([]string{"pauper"}, gear); got != "" {
		t.Fatalf("Rare gear rejected: %q", got)
	}
	gear[content.SlotRelic] = content.Gear{Name: "Epic relic", Rarity: content.RarityEpic}
	if got := abyssPactEquipmentError([]string{"pauper"}, gear); !strings.Contains(got, "Rare-or-lower") {
		t.Fatalf("Epic gear error = %q", got)
	}
	if got := abyssPactEquipmentError(nil, gear); got != "" {
		t.Fatalf("gear rejected without Pauper: %q", got)
	}
}

func TestAbyssPactRuleBounds(t *testing.T) {
	t.Parallel()

	pacts := []string{"anemic", "brittle", "deep_drums", "famine"}
	if got := abyssPactMaxHP(pacts, 101); got != 50 {
		t.Errorf("Anemic max HP = %d, want 50", got)
	}
	if got := abyssPactMaxHP(pacts, 1); got != 1 {
		t.Errorf("Anemic minimum max HP = %d, want 1", got)
	}
	if !abyssPactBossFloor(pacts, 3) || abyssPactBossFloor(pacts, 4) {
		t.Error("Deep Drums boss cadence is not every third floor")
	}
	if abyssPactAllowsRest(pacts) {
		t.Error("Famine allowed a rest floor")
	}
	if got := abyssPactDurabilityPasses(pacts); got != 2 {
		t.Errorf("Brittle durability passes = %d, want 2", got)
	}
}

func TestAbyssCursedHordeAddsOneBeneficialAffix(t *testing.T) {
	t.Parallel()

	mobs := []content.Mob{{Name: "First"}, {Name: "Second", Effects: []content.MobEffect{content.EffectEnraged}}}
	abyssApplyCursedHorde(mobs, fixedCombatRandom{intn: 0})
	if !mobHasEffect(mobs[0], content.EffectEnraged) {
		t.Fatalf("first mob effects = %v", mobs[0].Effects)
	}
	if len(mobs[1].Effects) != 2 || !mobHasEffect(mobs[1], content.EffectArmored) {
		t.Fatalf("second mob did not receive a distinct affix: %v", mobs[1].Effects)
	}
}
