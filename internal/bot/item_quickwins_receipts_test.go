package bot

import (
	"encoding/json"
	"testing"

	"ts3news/internal/content"
)

func TestForgeActualChangesUsePersistedOutcomes(t *testing.T) {
	before := json.RawMessage(`{"item":{"Stats":{"STR":99,"DEF":-100}}}`)
	after := json.RawMessage(`{"item":{"Stats":{"STR":100,"DEF":-102}}}`)
	got := forgeActualStatChanges(before, after)
	if len(got) != 2 || got[0].Before != 99 || got[0].After != 100 || got[0].Delta != 1 || got[1].Delta != -2 {
		t.Fatalf("receipt changes: %+v", got)
	}
	if got := forgeActualStatChanges(before, before); len(got) != 0 {
		t.Fatalf("failure invented changes: %+v", got)
	}
}

func TestItemQuickwinsTemperEndpointsKeepNegativeStatsTogether(t *testing.T) {
	gear := content.Gear{Slot: content.SlotHead, Stats: content.Stats{STR: 99, DEF: -100}}
	got, ok := exactForgeStatOutcome("temper", gear, .5)
	if !ok || got.SuccessStats == nil || got.FailureStats == nil {
		t.Fatal("temper endpoints unavailable")
	}
	if got.SuccessStats.STR != 100 || got.SuccessStats.DEF != -102 || got.FailureStats.STR != 99 || got.FailureStats.DEF != -100 {
		t.Fatalf("endpoints mixed extrema instead of actual outcomes: %+v", got)
	}
}

func TestItemQuickwinsPetInspectionHasNoBrokenInBonus(t *testing.T) {
	gear := content.Gear{Slot: content.SlotPet1, Stats: content.Stats{STR: 1000}, FoundAt: "2020-01-01T00:00:00Z"}
	var inspection itemInspectView
	if err := json.Unmarshal([]byte(toGearView(gear.Slot, gear).InspectJSON), &inspection); err != nil {
		t.Fatal(err)
	}
	if inspection.BrokenInAt != "" {
		t.Fatal("pet inspection offers an ineligible age bonus")
	}
	for _, stat := range inspection.BrokenInBonus {
		if stat.Value != 0 {
			t.Fatalf("pet inspection invented bonus: %+v", stat)
		}
	}
}

func TestItemQuickwinsLootRecommendationsClearStaleExplanations(t *testing.T) {
	rows := []runLootRow{
		{EscrowID: 1, Slot: "Head", IsUpgrade: true, StatPower: 20},
		{EscrowID: 2, Slot: "Head", IsUpgrade: true, StatPower: 10},
	}
	markAbyssBestRunLoot(rows)
	if rows[0].RunnerUpID != 2 || rows[0].BestReason == "" {
		t.Fatal("missing initial recommendation")
	}
	rows[0].IsUpgrade = false
	markAbyssBestRunLoot(rows)
	if rows[0].BestReason != "" || rows[0].RunnerUpID != 0 || rows[0].RunnerUpTitle != "" || !rows[1].CanEquipBest {
		t.Fatalf("stale recommendation survived: %+v", rows)
	}
}

func TestForgeActualChangesNeverRevealHiddenOrUnavailableItems(t *testing.T) {
	known := json.RawMessage(`{"item":{"Stats":{"STR":999}}}`)
	for _, hidden := range []json.RawMessage{json.RawMessage(`null`), json.RawMessage(`{}`), json.RawMessage(`{"item":{"unidentified":true,"Stats":{"STR":100}}}`)} {
		if len(forgeActualStatChanges(hidden, known)) != 0 || len(forgeActualStatChanges(known, hidden)) != 0 {
			t.Fatal("hidden or missing data leaked into receipt")
		}
	}
}
