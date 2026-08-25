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
