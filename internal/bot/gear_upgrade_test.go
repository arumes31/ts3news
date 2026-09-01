package bot

import (
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestGearShouldReplaceMatchesAutoEquipPriorities(t *testing.T) {
	current := content.Gear{
		Slot:         content.SlotHead,
		Rarity:       content.RarityLegendary,
		XPMultiplier: 1,
		Stats:        content.Stats{STR: 500, DEF: 500},
	}
	tests := []struct {
		name      string
		candidate content.Gear
		current   content.Gear
		want      bool
	}{
		{
			name: "higher XP multiplier wins before lower power",
			candidate: content.Gear{
				Slot: content.SlotHead, Rarity: content.RarityCommon,
				XPMultiplier: 2, Stats: content.Stats{STR: 1},
			},
			current: current,
			want:    true,
		},
		{
			name: "higher rarity replaces",
			candidate: content.Gear{
				Slot: content.SlotHead, Rarity: content.RarityDivine,
				XPMultiplier: 1, Stats: content.Stats{STR: 1},
			},
			current: current,
			want:    true,
		},
		{
			name: "higher combat rating replaces",
			candidate: content.Gear{
				Slot: content.SlotHead, Rarity: content.RarityLegendary,
				XPMultiplier: 1, Stats: content.Stats{STR: 2_000},
			},
			current: current,
			want:    true,
		},
		{
			name: "unidentified candidate is inert",
			candidate: content.Gear{
				Slot: content.SlotHead, Rarity: content.RarityDivine,
				XPMultiplier: 3, Stats: content.Stats{STR: 2_000}, Unidentified: true,
			},
			current: current,
			want:    false,
		},
		{
			name: "identified candidate replaces unidentified current",
			candidate: content.Gear{
				Slot: content.SlotHead, Rarity: content.RarityCommon,
			},
			current: content.Gear{
				Slot: content.SlotHead, Rarity: content.RarityDivine,
				XPMultiplier: 3, Stats: content.Stats{STR: 2_000}, Unidentified: true,
			},
			want: true,
		},
		{
			name:      "equal gear does not replace",
			candidate: current,
			current:   current,
			want:      false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := gearShouldReplace(test.candidate, test.current); got != test.want {
				t.Fatalf("gearShouldReplace() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestIsGearUpgradeUsesExactEquippedInstance(t *testing.T) {
	forgedCurrent := content.Gear{
		ID: "FORGED_HEAD", Slot: content.SlotHead, Rarity: content.RarityLegendary,
		XPMultiplier: 2, Stats: content.Stats{STR: 2_000, DEF: 2_000},
	}
	candidate := content.Gear{
		ID: "SHOP_HEAD", Slot: content.SlotHead, Rarity: content.RarityLegendary,
		XPMultiplier: 2, Stats: content.Stats{STR: 20, DEF: 20},
	}
	if isGearUpgrade(candidate, map[string]content.Gear{string(content.SlotHead): forgedCurrent}) {
		t.Fatal("base candidate was misbadged against the exact forged equipped instance")
	}
	if !isGearUpgrade(candidate, nil) {
		t.Fatal("candidate for an empty slot should be an upgrade")
	}
}

func TestAuctionUpgradeUsesExactListedInstance(t *testing.T) {
	listed := content.Gear{
		ID: "CUSTOM_LISTING", Name: "Forged Crown", Slot: content.SlotHead,
		Rarity: content.RarityLegendary, XPMultiplier: 2,
		Stats: content.Stats{STR: 2_000, DEF: 2_000},
	}
	payload, err := json.Marshal(listed)
	if err != nil {
		t.Fatal(err)
	}
	view := ahListingView{ItemType: "gear", ItemID: listed.ID}
	ahEnrichListing(&view, payload)

	current := listed
	current.Stats = content.Stats{STR: 20, DEF: 20}
	if !isGearUpgrade(view.gear, map[string]content.Gear{string(content.SlotHead): current}) {
		t.Fatal("custom listed instance stats were not used for its upgrade badge")
	}
}

func TestEquippedGearUpgradeIndexReconstructsPersistedInstance(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	forged := content.Gear{
		ID: "CUSTOM_FORGED_HEAD", Name: "Forged Crown", Slot: content.SlotHead,
		Rarity: content.RarityLegendary, XPMultiplier: 2,
		Stats: content.Stats{STR: 2_000, DEF: 2_000},
	}
	payload, err := json.Marshal(forged)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT slot, gear_id, item_data FROM user_gear").
		WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"slot", "gear_id", "item_data"}).
			AddRow(string(content.SlotHead), forged.ID, string(payload)))

	got := (&Bot{DB: database}).equippedGearUpgradeIndex("player")[string(content.SlotHead)]
	if got.ID != forged.ID || got.Stats != forged.Stats || got.XPMultiplier != forged.XPMultiplier {
		t.Fatalf("equipped instance = %+v, want persisted forged instance %+v", got, forged)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestShouldEquipComparesExactPersistedCurrentInstance(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	forgedCurrent := content.Gear{
		ID: "CUSTOM_FORGED_HEAD", Name: "Forged Crown", Slot: content.SlotHead,
		Rarity: content.RarityLegendary, XPMultiplier: 2,
		Stats: content.Stats{STR: 2_000, DEF: 2_000},
	}
	payload, err := json.Marshal(forgedCurrent)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_gear").
		WithArgs("player", string(content.SlotHead)).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).
			AddRow(forgedCurrent.ID, string(payload)))

	candidate := forgedCurrent
	candidate.ID = "BASE_SHOP_HEAD"
	candidate.Stats = content.Stats{STR: 20, DEF: 20}
	if (&Bot{DB: database}).shouldEquip("player", candidate) {
		t.Fatal("auto-equip flattened the exact forged current instance to catalog data")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFeaturedShopUpgradeUsesExactEquippedInstance(t *testing.T) {
	const seed = int64(42)
	featured := content.FeaturedShopItem(seed)
	forgedCurrent := featured
	forgedCurrent.Stats = content.Stats{STR: 100_000, DEF: 100_000, INT: 100_000}

	stock := stockForSeed(seed, map[string]content.Gear{string(featured.Slot): forgedCurrent})
	if len(stock) == 0 || !stock[0].Featured {
		t.Fatal("featured relic is not pinned to the first shop card")
	}
	if stock[0].IsUpgrade {
		t.Fatal("featured relic was misbadged against the exact forged equipped instance")
	}
}

func TestAHUpgradeCountUsesExactListedInstance(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	listed := content.Gear{
		ID: "CUSTOM_LISTING", Name: "Forged Crown", Slot: content.SlotHead,
		Rarity: content.RarityLegendary, XPMultiplier: 2,
		Stats: content.Stats{STR: 2_000, DEF: 2_000},
	}
	payload, err := json.Marshal(listed)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT item_type, item_id, item_data").
		WithArgs(false).
		WillReturnRows(sqlmock.NewRows([]string{"item_type", "item_id", "item_data"}).
			AddRow("gear", listed.ID, string(payload)))

	current := listed
	current.Stats = content.Stats{STR: 20, DEF: 20}
	count := (&Bot{DB: database}).ahActiveListingsCount(
		"",
		map[string]content.Gear{string(content.SlotHead): current},
		true,
		false,
	)
	if count != 1 {
		t.Fatalf("upgrade count = %d, want 1 for the exact forged listing", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
