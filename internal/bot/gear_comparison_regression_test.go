package bot

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"ts3news/internal/content"
)

func TestGearComparisonCountsAllMeasuredContributions(t *testing.T) {
	current := content.Gear{Slot: content.SlotHead, Rarity: content.RarityRare, XPMultiplier: 1}
	candidate := current
	candidate.Stats.STR = 1
	candidate.XPMultiplier = 1.1
	candidate.RegenAmount, candidate.RegenIntervalSec = 1, 1
	candidate.MaxDurability, candidate.Sockets, candidate.Insured = 1, 1, true
	forward, reverse := compareGear(candidate, current, true), compareGear(current, candidate, true)
	if forward.Gains != 6 || forward.Losses != 0 || forward.Status != "upgrade" {
		t.Fatalf("incomplete improvement counts: %+v", forward)
	}
	if reverse.Gains != 0 || reverse.Losses != 6 || reverse.Status != "downgrade" {
		t.Fatalf("incomplete loss counts: %+v", reverse)
	}
}

func TestGearUpgradeRejectsTradeoffs(t *testing.T) {
	current := content.Gear{Slot: content.SlotHead, Rarity: content.RarityRare, XPMultiplier: 1.1, Stats: content.Stats{STR: 100, DEF: 100, MNA: 100}}
	for _, test := range []struct {
		name   string
		change func(*content.Gear)
	}{
		{"rarity cannot compensate for lost stats", func(g *content.Gear) { g.Rarity = content.RarityEternal; g.Stats.STR = 1 }},
		{"XP cannot compensate for lost stats", func(g *content.Gear) { g.XPMultiplier = 2; g.Stats.STR = 1 }},
		{"offense cannot silently replace defense", func(g *content.Gear) { g.Stats.STR = 1000; g.Stats.DEF = 1 }},
		{"mana loss counts", func(g *content.Gear) { g.Stats.STR++; g.Stats.MNA = 0 }},
		{"XP loss counts", func(g *content.Gear) { g.Stats.STR++; g.XPMultiplier = 1 }},
		{"special changes need review", func(g *content.Gear) { g.Stats.STR++; g.Special = content.EffectFragile }},
		{"rarity alone is not a gain", func(g *content.Gear) { g.Rarity++ }},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := current
			test.change(&candidate)
			if gearShouldReplace(candidate, current) {
				t.Fatal("tradeoff was recommended as an automatic upgrade")
			}
		})
	}
}

func TestGearUpgradeUsesEffectiveXPAndEveryCombatStat(t *testing.T) {
	current := content.Gear{Slot: content.SlotNeck, Rarity: content.RarityRare, XPMultiplier: 1.3, Stats: content.Stats{STR: 100, MNA: 100}}
	candidate := current
	candidate.XPMultiplier = 1.02
	candidate.Stats.MNA++
	if !gearShouldReplace(candidate, current) {
		t.Fatal("mana improvement with identical effective XP should upgrade")
	}
	if gearShouldReplace(current, candidate) {
		t.Fatal("replacement rule must not recommend both directions")
	}
}

func TestEmptySlotUpgradeRejectsNegativeStats(t *testing.T) {
	candidate := content.Gear{Slot: content.SlotHead, Stats: content.Stats{HP: -100, STR: 500}}
	if isGearUpgrade(candidate, nil) {
		t.Fatal("empty slot hid an HP loss")
	}
	hidden := map[string]content.Gear{"Head": {Slot: content.SlotHead, Unidentified: true}}
	if isGearUpgrade(candidate, hidden) {
		t.Fatal("inert unidentified item hid an HP loss")
	}
}

func TestShopComparisonUsesGameplayXP(t *testing.T) {
	current := content.Gear{Slot: content.SlotNeck, Rarity: content.RarityRare, XPMultiplier: 1.02}
	candidate := current
	candidate.XPMultiplier = 1.3
	comparison := shopGearComparison(candidate, map[string]content.Gear{string(current.Slot): current})
	if comparison.XPBonusDelta != 0 {
		t.Fatalf("capped XP delta = %g, want 0", comparison.XPBonusDelta)
	}
	if view := toGearView(candidate.Slot, candidate); view.XPBonusPct != 2 {
		t.Fatalf("displayed XP = %d, want 2", view.XPBonusPct)
	}
	common := content.Gear{Slot: content.SlotHead, Rarity: content.RarityCommon, XPMultiplier: 2}
	if view := toGearView(common.Slot, common); view.XPBonusPct != 0 {
		t.Fatalf("common gear XP = %d, want 0", view.XPBonusPct)
	}
}

func TestShopComparisonEmptySlotIncludesAllGains(t *testing.T) {
	candidate := content.Gear{Slot: content.SlotHead, Rarity: content.RarityRare, XPMultiplier: 1.1, Stats: content.Stats{STR: 5}, Special: content.EffectQuick}
	comparison := shopGearComparison(candidate, nil)
	if comparison.XPBonusDelta != 10 || len(comparison.Stats) != 1 || len(comparison.AddedSpecials) != 1 {
		t.Fatalf("incomplete empty slot comparison: %+v", comparison)
	}
	candidate.Unidentified = true
	comparison = shopGearComparison(candidate, nil)
	if !comparison.Unknown || comparison.CRDelta != 0 || comparison.ScoreDelta != 0 || len(comparison.Stats) != 0 {
		t.Fatalf("hidden roll leaked: %+v", comparison)
	}
}

func TestShouldEquipFailsClosedOnDatabaseError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_gear").WithArgs("player", "Head").WillReturnError(errors.New("database unavailable"))
	candidate := content.Gear{Slot: content.SlotHead, Stats: content.Stats{STR: 100}}
	if (&Bot{DB: database}).shouldEquip("player", candidate) {
		t.Fatal("database error was treated as an empty slot")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAutoListPreservesStatTradeoffAgainstExactCurrentItem(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	current := content.Gear{ID: "FORGED", Slot: content.SlotHead, Rarity: content.RarityEternal, Stats: content.Stats{STR: 100, DEF: 10}, XPMultiplier: 1.1}
	payload, err := json.Marshal(current)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_gear").WithArgs("player", "Head").WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow(current.ID, string(payload)))
	candidate := current
	candidate.Rarity = content.RarityRare
	candidate.Stats = content.Stats{STR: 10, DEF: 100}
	if (&Bot{DB: database}).autoListUnwantedItems("player", candidate) {
		t.Fatal("useful defensive alternative was auto-listed")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunLootComparisonRefreshesAfterEquipmentChange(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	candidate := content.Gear{ID: "OLD_DROP", Slot: content.SlotHead, Rarity: content.RarityEternal, Stats: content.Stats{STR: 20, DEF: 10}, XPMultiplier: 1.3}
	current := content.Gear{ID: "NEW_EQUIPMENT", Slot: content.SlotHead, Rarity: content.RarityRare, Stats: content.Stats{STR: 100, DEF: 1}, XPMultiplier: 1.1}
	grant, err := json.Marshal(abyssLootGrant{Type: "gear", Gear: &candidate})
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT id, label, depth, item_type, item_data, equip_on_bank").WithArgs("player").WillReturnRows(sqlmock.NewRows([]string{"id", "label", "depth", "item_type", "item_data", "equip_on_bank"}).AddRow(1, "Old drop [color=#41c97a]▲ empty slot[/color]", 1, "gear", grant, false))
	rows := (&Bot{DB: database}).currentRunLootManifest("player", map[content.GearSlot]content.Gear{content.SlotHead: current}, nil)
	if len(rows) != 1 {
		t.Fatalf("got %d loot rows", len(rows))
	}
	row := rows[0]
	if row.IsUpgrade || row.CanEquipBest || row.EmptySlot {
		t.Fatalf("stale upgrade survived equipment change: %+v", row)
	}
	changes := map[string]int{}
	for _, stat := range row.StatChanges {
		changes[stat.Label] = stat.Value
	}
	if changes["STR"] != -80 || changes["DEF"] != 9 || row.XPBonusDelta != 20 {
		t.Fatalf("incorrect fresh changes: %+v", row)
	}
	if strings.Contains(string(row.Label), "empty slot") || strings.Contains(row.Title, "empty slot") {
		t.Fatal("stale drop-time comparison is still visible")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
