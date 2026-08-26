package bot

import (
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestNormalizeAbyssLootSettingsKeepsSafeChoices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input abyssLootSettings
		want  abyssLootSettings
	}{
		{abyssLootSettings{TargetCategory: " WEAPON ", AutoSalvageMax: 0}, abyssLootSettings{TargetCategory: "weapon", AutoSalvageMax: 0}},
		{abyssLootSettings{TargetCategory: "armor", AutoSalvageMax: int(content.RarityUncommon)}, abyssLootSettings{TargetCategory: "armor", AutoSalvageMax: 1}},
		{abyssLootSettings{TargetCategory: "invalid", AutoSalvageMax: 99}, abyssLootSettings{AutoSalvageMax: 1}},
		{abyssLootSettings{TargetCategory: "jewelry", AutoSalvageMax: -10}, abyssLootSettings{TargetCategory: "jewelry", AutoSalvageMax: -1}},
		{abyssLootSettings{AutoSalvageMax: -1, DuplicateLegendConvert: true}, abyssLootSettings{AutoSalvageMax: -1, DuplicateLegendConvert: true}},
	}
	for _, test := range tests {
		if got := normalizeAbyssLootSettings(test.input); got != test.want {
			t.Errorf("normalize(%#v) = %#v, want %#v", test.input, got, test.want)
		}
	}
}

func TestAbyssReservedLootIDsDropsInvalidEntries(t *testing.T) {
	t.Parallel()

	ids := abyssReservedLootIDs(map[int64]bool{-1: true, 0: true, 7: true, 11: false})
	if len(ids) != 1 || ids[0] != 7 {
		t.Fatalf("reserved ids = %v, want active positive entries only", ids)
	}
}

func TestAbyssMaterialFlowTracksSourcesAndSinksSeparately(t *testing.T) {
	uid := t.Name()
	before := abyssMaterialFlow(uid)
	beforeSource := before["core"]["source"]
	beforeSink := before["core"]["sink"]
	recordAbyssMaterialFlow(uid, "core", "source", 7)
	recordAbyssMaterialFlow(uid, "core", "sink", 3)
	recordAbyssMaterialFlow(uid, "core", "invalid", 99)
	flow := abyssMaterialFlow(uid)
	if flow["core"]["source"]-beforeSource != 7 || flow["core"]["sink"]-beforeSink != 3 {
		t.Fatalf("material flow = %#v", flow)
	}
}

func TestAbyssPresetValidationAndNames(t *testing.T) {
	t.Parallel()

	if _, ok := normalizeAbyssPresetSlot(0); ok {
		t.Fatal("slot zero accepted")
	}
	if slot, ok := normalizeAbyssPresetSlot(3); !ok || slot != "3" {
		t.Fatalf("slot 3 = (%q, %v)", slot, ok)
	}
	if got := normalizeAbyssPresetName("", 2); got != "Loadout 2" {
		t.Fatalf("default preset name = %q", got)
	}
	if got := normalizeAbyssPresetName(strings.Repeat("x", 40), 1); len(got) != 32 {
		t.Fatalf("long preset name length = %d", len(got))
	}
}

func TestAbyssSmartLootTargetSlotsPrefersEmptyThenWeakest(t *testing.T) {
	t.Parallel()
	equipped := map[content.GearSlot]content.Gear{
		content.SlotHead:  {Slot: content.SlotHead, Rarity: content.RarityCommon, Stats: content.Stats{DEF: 10}},
		content.SlotChest: {Slot: content.SlotChest, Rarity: content.RarityCommon, Stats: content.Stats{DEF: 30}},
	}

	targets, reason := abyssSmartLootTargetSlots(equipped, []content.GearSlot{content.SlotHead, content.SlotChest, content.SlotFeet}, "armor")
	if reason != abyssSmartLootEmpty || len(targets) != 1 || targets[0] != content.SlotFeet {
		t.Fatalf("empty targets = %v (%q), want Feet", targets, reason)
	}

	equipped[content.SlotFeet] = content.Gear{Slot: content.SlotFeet, Rarity: content.RarityCommon, Stats: content.Stats{DEF: 10}}
	targets, reason = abyssSmartLootTargetSlots(equipped, []content.GearSlot{content.SlotHead, content.SlotChest, content.SlotFeet}, "armor")
	if reason != abyssSmartLootWeakest || len(targets) != 2 || targets[0] != content.SlotHead || targets[1] != content.SlotFeet {
		t.Fatalf("weakest targets = %v (%q), want Head and Feet", targets, reason)
	}
}

func TestAbyssSmartLootTargetSlotsHonorsCategory(t *testing.T) {
	t.Parallel()
	targets, reason := abyssSmartLootTargetSlots(nil, []content.GearSlot{content.SlotHead, content.SlotMainHand, content.SlotFinger1}, "jewelry")
	if reason != abyssSmartLootEmpty || len(targets) != 1 || targets[0] != content.SlotFinger1 {
		t.Fatalf("category targets = %v (%q), want Finger1", targets, reason)
	}
}

func TestApplyAbyssSmartLootRespectsChanceAndRarityOutcome(t *testing.T) {
	t.Parallel()
	original := content.Gear{ID: "original", Slot: content.SlotHead, Rarity: content.RarityEternal}
	unchanged, reason := applyAbyssSmartLoot(original, content.GearDropPoolStarter, nil, nil, "", abyssSmartLootChance)
	if unchanged.ID != original.ID || reason != "" {
		t.Fatalf("failed chance gate changed drop: %#v (%q)", unchanged, reason)
	}

	biased, reason := applyAbyssSmartLoot(original, content.GearDropPoolStarter, nil, nil, "", 0)
	if reason != abyssSmartLootEmpty || biased.Rarity != content.RarityEternal {
		t.Fatalf("biased drop = %#v (%q), want empty-slot reason and unchanged Eternal rarity", biased, reason)
	}
}

func TestAbyssSetPityCandidatesRequireExactlyThreeDistinctPieces(t *testing.T) {
	t.Parallel()
	owned := map[string]bool{
		"ABYSS_BERSERKER_RING":  true,
		"ABYSS_FIREBRAND_SWORD": true,
		"ABYSS_ZEPHYR_DAGGER":   true,
	}

	candidates := abyssSetPityCandidates(owned, "")
	if len(candidates) != 3 {
		t.Fatalf("candidate count = %d, want 3 missing Predator pieces", len(candidates))
	}
	for _, gear := range candidates {
		if gear.SetID != "predator" || owned[gear.ID] {
			t.Fatalf("invalid completion candidate: %#v", gear)
		}
	}

	owned["ABYSS_ASSASSIN_HOOD"] = true
	if got := abyssSetPityCandidates(owned, ""); len(got) != 0 {
		t.Fatalf("four owned pieces still produced candidates: %#v", got)
	}

	harvester := map[string]bool{
		"ABYSS_HARVESTER_HOOK": true,
		"ABYSS_HARVESTER_EYE":  true,
		"ABYSS_HARVESTER_BAND": true,
	}
	if got := abyssSetPityCandidates(harvester, ""); len(got) != 0 {
		t.Fatalf("three-piece set produced fourth-piece candidates: %#v", got)
	}
}

func TestMarkAbyssEscrowedGearOwnedUsesStructuredGrant(t *testing.T) {
	t.Parallel()
	owned := map[string]bool{}
	data := []byte(`{"type":"gear","gear":{"ID":"ABYSS_ASSASSIN_HOOD"}}`)
	if !markAbyssEscrowedGearOwned(owned, data) || !owned["ABYSS_ASSASSIN_HOOD"] {
		t.Fatalf("escrowed gear was not included in ownership: %#v", owned)
	}
	if markAbyssEscrowedGearOwned(owned, []byte(`{"type":"gold","gold":10}`)) {
		t.Fatal("non-gear grant changed ownership")
	}
	if markAbyssEscrowedGearOwned(owned, []byte(`not-json`)) {
		t.Fatal("invalid grant changed ownership")
	}
}

func TestApplyAbyssSetPityPreservesBoundariesAndRarity(t *testing.T) {
	t.Parallel()
	owned := map[string]bool{
		"ABYSS_BERSERKER_RING":  true,
		"ABYSS_FIREBRAND_SWORD": true,
		"ABYSS_ZEPHYR_DAGGER":   true,
	}
	original := content.Gear{ID: "original", Slot: content.SlotChest, Rarity: content.RarityEternal}

	unchanged, setID := applyAbyssSetPity(
		original,
		content.GearDropPoolAbyss,
		owned,
		"",
		abyssSetPityChance,
	)
	if unchanged.ID != original.ID || setID != "" {
		t.Fatalf("failed chance gate changed drop: %#v (%q)", unchanged, setID)
	}

	biased, setID := applyAbyssSetPity(
		original,
		content.GearDropPoolAbyss,
		owned,
		"",
		0,
	)
	if setID != "predator" || biased.ID == original.ID || biased.Rarity != content.RarityEternal {
		t.Fatalf("biased drop = %#v (%q), want missing Predator piece with Eternal rarity", biased, setID)
	}

	standard, setID := applyAbyssSetPity(
		original,
		content.GearDropPoolStandard,
		owned,
		"",
		0,
	)
	if standard.ID != original.ID || setID != "" {
		t.Fatalf("standard pool changed drop: %#v (%q)", standard, setID)
	}

	categoryOwned := map[string]bool{
		"ABYSS_BERSERKER_RING":  true,
		"ABYSS_HELLFIRE_RING":   true,
		"ABYSS_FIREBRAND_SWORD": true,
	}
	categoryDrop, setID := applyAbyssSetPity(
		original,
		content.GearDropPoolAbyss,
		categoryOwned,
		"jewelry",
		0,
	)
	if categoryDrop.ID != original.ID || setID != "" {
		t.Fatalf("category chase was overridden: %#v (%q)", categoryDrop, setID)
	}
}
