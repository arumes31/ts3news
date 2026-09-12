package bot

import (
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"math"
	"strings"
	"testing"
	"time"

	"ts3news/internal/content"
)

func TestGearComparisonIncludesBrokenInContributions(t *testing.T) {
	old := content.Gear{Slot: content.SlotHead, Stats: content.Stats{STR: 1000}, FoundAt: time.Now().Add(-31 * 24 * time.Hour).Format(time.RFC3339)}
	candidate := content.Gear{Slot: content.SlotHead, Stats: content.Stats{STR: 1001}}
	got := compareGear(candidate, old, true)
	if got.IsUpgrade || got.Status != "downgrade" || got.Changes[0].Before != 1010 || got.Changes[0].Delta != -9 {
		t.Fatalf("lost broken-in bonus: %+v", got)
	}
	old.Slot, candidate.Slot = content.SlotPet1, content.SlotPet1
	if got = compareGear(candidate, old, true); !got.IsUpgrade || got.Changes[0].Before != 1000 {
		t.Fatalf("pet received a player-only bonus: %+v", got)
	}
}

func TestGearComparisonEmptySlotRequiresManualHarmfulEffects(t *testing.T) {
	for _, change := range []func(*content.Gear){func(g *content.Gear) { g.Cursed = true }, func(g *content.Gear) { g.Eldritch = true }, func(g *content.Gear) { g.Doomed = true }, func(g *content.Gear) { g.Special = content.EffectFragile }} {
		g := content.Gear{Slot: content.SlotHead, Stats: content.Stats{STR: 100}}
		change(&g)
		if got := compareGear(g, content.Gear{}, false); got.IsUpgrade || got.Status != "tradeoff" {
			t.Fatalf("harmful empty-slot recommendation: %+v", got)
		}
	}
	good := content.Gear{Slot: content.SlotHead, Stats: content.Stats{STR: 100}, Special: content.EffectQuick}
	if got := compareGear(good, content.Gear{}, false); !got.IsUpgrade {
		t.Fatalf("safe empty-slot gain rejected: %+v", got)
	}
}

func TestGearComparisonMalformedSavedStatsNeverUseCatalogFallback(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery("SELECT slot, gear_id, item_data FROM user_gear").WithArgs("player").WillReturnRows(sqlmock.NewRows([]string{"slot", "gear_id", "item_data"}).AddRow("Head", "B_Head", `{"Stats":{"STR":"bad"}}`))
	got := shopGearComparison(content.Gear{Slot: content.SlotHead, Stats: content.Stats{STR: 9999}}, (&Bot{DB: db}).equippedGearUpgradeIndex("player"))
	if !got.Unknown || got.IsUpgrade {
		t.Fatalf("catalog fallback hid corrupt roll: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGearComparisonStatusesAndDetails(t *testing.T) {
	old := content.Gear{Slot: content.SlotHead, Rarity: content.RarityRare, XPMultiplier: 1.1, Stats: content.Stats{STR: 100, DEF: 20}}
	for _, tc := range []struct {
		name   string
		stats  content.Stats
		status string
	}{
		{"upgrade", content.Stats{STR: 110, DEF: 20}, "upgrade"},
		{"tradeoff", content.Stats{STR: 110, DEF: 10}, "tradeoff"},
		{"downgrade", content.Stats{STR: 90, DEF: 20}, "downgrade"},
		{"equivalent", old.Stats, "equivalent"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate := old
			candidate.Stats = tc.stats
			got := compareGear(candidate, old, true)
			if got.Status != tc.status || len(got.Reasons) == 0 {
				t.Fatalf("comparison: %+v", got)
			}
			if tc.status == "upgrade" {
				row := got.Changes[0]
				if row.Before != 100 || row.After != 110 || row.Delta != 10 || row.Percent != 10 || !row.HasPercent {
					t.Fatalf("change: %+v", row)
				}
			}
		})
	}
}

func TestGearComparisonPropertySafeguards(t *testing.T) {
	base := content.Gear{Slot: content.SlotHead, Rarity: content.RarityRare, XPMultiplier: 1.1, Stats: content.Stats{STR: 100}, Sockets: 2, MaxDurability: 80, Insured: true, Attuned: true}
	for _, tc := range []struct {
		name   string
		change func(*content.Gear)
		reason string
	}{
		{"socket capacity", func(g *content.Gear) { g.Sockets = 1 }, "Socket capacity decreases"},
		{"durability", func(g *content.Gear) { g.MaxDurability = 79 }, "Maximum durability decreases"},
		{"insurance", func(g *content.Gear) { g.Insured = false }, "insurance"},
		{"attunement", func(g *content.Gear) { g.Attuned = false }, "Attunement"},
		{"rune", func(g *content.Gear) { g.Rune = "Fire" }, "Rune"},
		{"element", func(g *content.Gear) { g.Element = content.ElementFire }, "Element"},
		{"set", func(g *content.Gear) { g.SetID = "warden" }, "Set membership"},
		{"curse", func(g *content.Gear) { g.Cursed = true }, "properties change"},
		{"eldritch", func(g *content.Gear) { g.Eldritch = true }, "properties change"},
		{"doom", func(g *content.Gear) { g.Doomed = true }, "properties change"},
		{"effect", func(g *content.Gear) { g.Special = content.EffectFragile }, "Special effects"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate := base
			candidate.Stats.STR++
			tc.change(&candidate)
			got := compareGear(candidate, base, true)
			if got.IsUpgrade || got.Status != "tradeoff" || !strings.Contains(strings.Join(got.Reasons, " "), tc.reason) {
				t.Fatalf("unexplained property loss: %+v", got)
			}
		})
	}
}

func TestGearComparisonNegativeAndZeroBaselines(t *testing.T) {
	old := content.Gear{Slot: content.SlotHead, Stats: content.Stats{HP: -100}}
	candidate := old
	candidate.Stats.HP = -90
	candidate.Stats.MNA = 1
	candidate.Stats.CHA = 9
	got := compareGear(candidate, old, true)
	if got.Gains != 2 || got.Losses != 0 || !got.IsUpgrade {
		t.Fatalf("gains: %+v", got)
	}
	if got.Changes[0].Percent != 10 || !got.Changes[0].HasPercent || got.Changes[1].HasPercent || got.Changes[2].Combat {
		t.Fatalf("baselines or flavour: %+v", got.Changes)
	}
}

func TestGearComparisonRarityOnlyIsEquivalent(t *testing.T) {
	old := content.Gear{Slot: content.SlotHead, Rarity: content.RarityRare, XPMultiplier: 1.1, Stats: content.Stats{STR: 10}}
	candidate := old
	candidate.Rarity = content.RarityEternal
	got := compareGear(candidate, old, true)
	if got.Status != "equivalent" || got.PowerDelta != 0 || got.CRDelta <= 0 || got.IsUpgrade {
		t.Fatalf("rarity premium misclassified: %+v", got)
	}
}

func TestGearComparisonUnknownOnEquipmentReadFailure(t *testing.T) {
	for _, partial := range []bool{false, true} {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		query := mock.ExpectQuery("SELECT slot, gear_id, item_data FROM user_gear").WithArgs("player")
		if partial {
			query.WillReturnRows(sqlmock.NewRows([]string{"slot", "gear_id", "item_data"}).AddRow("Head", "B_Head", nil).AddRow("Neck", "B_Neck", nil).RowError(1, errors.New("partial read")))
		} else {
			query.WillReturnError(errors.New("unavailable"))
		}
		index := (&Bot{DB: db}).equippedGearUpgradeIndex("player")
		g := content.Gear{Slot: content.SlotHead, Stats: content.Stats{STR: 999}}
		got := shopGearComparison(g, index)
		if !got.Unknown || isGearUpgrade(g, index) || !strings.Contains(strings.Join(got.Reasons, " "), "loaded") {
			t.Fatalf("failed read looked empty: %+v", got)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
		_ = db.Close()
	}
}

func TestRunLootRankingUsesPowerThenXPThenRegenThenID(t *testing.T) {
	rows := []runLootRow{
		{EscrowID: 9, Slot: "Head", IsUpgrade: true, StatPower: 100, EffectiveXP: 1.1, CR: 9999},
		{EscrowID: 8, Slot: "Head", IsUpgrade: true, StatPower: 101, EffectiveXP: 1.1},
		{EscrowID: 7, Slot: "Head", IsUpgrade: true, StatPower: 101, EffectiveXP: 1.2},
		{EscrowID: 6, Slot: "Head", IsUpgrade: true, StatPower: 101, EffectiveXP: 1.2, RegenRate: 1},
		{EscrowID: 5, Slot: "Head", IsUpgrade: true, StatPower: 101, EffectiveXP: 1.2, RegenRate: 1},
	}
	markAbyssBestRunLoot(rows)
	for _, row := range rows {
		if row.CanEquipBest != (row.EscrowID == 5) {
			t.Fatalf("inconsistent winner: %+v", rows)
		}
	}
}

func TestGearComparisonRejectsInvalidXP(t *testing.T) {
	for _, xp := range []float64{math.NaN(), math.Inf(1), -1} {
		g := content.Gear{Slot: content.SlotHead, Rarity: content.RarityRare, XPMultiplier: xp, Stats: content.Stats{STR: 10}}
		if got := compareGear(g, content.Gear{}, false); got.Status != "unknown" || got.IsUpgrade {
			t.Fatalf("invalid XP accepted: %+v", got)
		}
	}
}

func TestGearComparisonUsesRegenRatesAndGemMultisets(t *testing.T) {
	old := content.Gear{Slot: content.SlotHead, Stats: content.Stats{STR: 10}, RegenAmount: 10, RegenIntervalSec: 10, Gemstones: []string{"Ruby", "Sapphire", "Ruby"}}
	candidate := old
	candidate.RegenAmount = 2
	candidate.RegenIntervalSec = 1
	candidate.Gemstones = []string{"Ruby", "Ruby", "Sapphire"}
	if got := compareGear(candidate, old, true); !got.IsUpgrade || got.RegenDelta != 1 {
		t.Fatalf("equivalent gems and faster regen: %+v", got)
	}
	candidate.Gemstones = []string{"Ruby", "Sapphire"}
	if got := compareGear(candidate, old, true); got.IsUpgrade {
		t.Fatal("duplicate gem loss ignored")
	}
}
