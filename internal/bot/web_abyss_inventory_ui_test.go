package bot

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestAbyssSetDisplayMax(t *testing.T) {
	tests := map[string]int{
		"predator":  6,
		"warden":    6,
		"harvester": 3,
		"unknown":   0,
	}
	for setID, want := range tests {
		if got := abyssSetDisplayMax(setID); got != want {
			t.Errorf("abyssSetDisplayMax(%q) = %d, want %d", setID, got, want)
		}
	}
}

func TestMarkAbyssBestRunLoot(t *testing.T) {
	rows := []runLootRow{
		{EscrowID: 1, Slot: "weapon", CR: 120, CRDelta: 20},
		{EscrowID: 2, Slot: "weapon", CR: 150, CRDelta: 50},
		{EscrowID: 3, Slot: "armor", CR: 80, CRDelta: -10},
		{EscrowID: 4, Slot: "ring", CR: 25, CRDelta: 25},
		{EscrowID: 5, ItemType: "gold"},
	}

	markAbyssBestRunLoot(rows)

	for i := range rows {
		want := rows[i].EscrowID == 2 || rows[i].EscrowID == 4
		if rows[i].CanEquipBest != want {
			t.Errorf("row %d CanEquipBest = %t, want %t", rows[i].EscrowID, rows[i].CanEquipBest, want)
		}
	}
}

func TestAbyssRunLootJunkQuoteProtectsSpecialDrops(t *testing.T) {
	ordinary := content.Gear{
		ID: "ordinary", Name: "Ordinary Blade", Slot: content.SlotMainHand,
		Rarity: content.RarityCommon, MaxDurability: 20, Stats: content.Stats{STR: 12},
	}
	wantValue := max(gearPrice(ordinary)/2, int64(1))
	tests := []struct {
		name  string
		grant abyssLootGrant
		ok    bool
	}{
		{name: "common ordinary", grant: abyssLootGrant{Type: "gear", Gear: &ordinary}, ok: true},
		{name: "uncommon ordinary", grant: abyssLootGrant{Type: "gear", Gear: func() *content.Gear { g := ordinary; g.Rarity = content.RarityUncommon; return &g }()}, ok: true},
		{name: "rare", grant: abyssLootGrant{Type: "gear", Gear: func() *content.Gear { g := ordinary; g.Rarity = content.RarityRare; return &g }()}},
		{name: "unidentified", grant: abyssLootGrant{Type: "gear", Gear: func() *content.Gear { g := ordinary; g.Unidentified = true; return &g }()}},
		{name: "wishlist", grant: abyssLootGrant{Type: "gear", Gear: &ordinary, Wishlist: true}},
		{name: "set pity", grant: abyssLootGrant{Type: "gear", Gear: &ordinary, SetPity: true}},
		{name: "smart loot", grant: abyssLootGrant{Type: "gear", Gear: &ordinary, SmartLoot: true}},
		{name: "set piece", grant: abyssLootGrant{Type: "gear", Gear: func() *content.Gear { g := ordinary; g.SetID = "warden"; return &g }()}},
		{name: "affixed", grant: abyssLootGrant{Type: "gear", Gear: func() *content.Gear { g := ordinary; g.Special = content.EffectLucky; return &g }()}},
		{name: "socketed", grant: abyssLootGrant{Type: "gear", Gear: func() *content.Gear { g := ordinary; g.Sockets = 1; return &g }()}},
		{name: "tempered", grant: abyssLootGrant{Type: "gear", Gear: func() *content.Gear { g := ordinary; g.Temper = 1; return &g }()}},
		{name: "corrupted", grant: abyssLootGrant{Type: "gear", Gear: func() *content.Gear { g := ordinary; g.Corrupted = true; return &g }()}},
		{name: "appearance", grant: abyssLootGrant{Type: "gear", Gear: func() *content.Gear { g := ordinary; g.AppearanceID = "skin"; return &g }()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, ok := abyssRunLootJunkQuote(tt.grant)
			if ok != tt.ok {
				t.Fatalf("eligible = %t, want %t (value %d)", ok, tt.ok, value)
			}
			if tt.ok && value < wantValue {
				t.Fatalf("value = %d, want at least common quote %d", value, wantValue)
			}
			if !tt.ok && value != 0 {
				t.Fatalf("protected item value = %d, want 0", value)
			}
		})
	}
}

func TestHandleAbyssSellJunkLootCommitsDeleteAndCacheCredit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	gear := content.Gear{ID: "junk", Slot: content.SlotMainHand, Rarity: content.RarityCommon, Stats: content.Stats{STR: 10}}
	grant := mustAbyssLootGrantJSON(t, abyssLootGrant{Type: "gear", Gear: &gear})
	value, ok := abyssRunLootJunkQuote(abyssLootGrant{Type: "gear", Gear: &gear})
	if !ok {
		t.Fatal("ordinary test gear was not eligible")
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT downed FROM abyss_active").WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"downed"}).AddRow(false))
	mock.ExpectQuery("SELECT item_type,item_data,equip_on_bank").WithArgs(int64(31), "player").
		WillReturnRows(sqlmock.NewRows([]string{"item_type", "item_data", "equip_on_bank"}).AddRow("gear", grant, false))
	mock.ExpectExec("DELETE FROM abyss_escrow_loot").WithArgs(int64(31), "player").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("UPDATE abyss_active SET escrow=escrow").WithArgs(value, "player").
		WillReturnRows(sqlmock.NewRows([]string{"escrow"}).AddRow(777))
	mock.ExpectCommit()
	body, err := json.Marshal(map[string]any{"id": 31, "quoted_gold": value})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/abyss/loot/sell_junk", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	(&WebServer{bot: &Bot{DB: db}}).handleAbyssSellJunkLoot(rec, req, "player")
	var response struct {
		OK     bool  `json:"ok"`
		Value  int64 `json:"value"`
		Escrow int64 `json:"escrow"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.OK || response.Value != value || response.Escrow != 777 {
		t.Fatalf("response = %+v", response)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAbyssSellJunkLootRejectsStaleQuoteWithoutMutation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	gear := content.Gear{ID: "junk", Slot: content.SlotMainHand, Rarity: content.RarityCommon, Stats: content.Stats{STR: 10}}
	grant := mustAbyssLootGrantJSON(t, abyssLootGrant{Type: "gear", Gear: &gear})
	value, _ := abyssRunLootJunkQuote(abyssLootGrant{Type: "gear", Gear: &gear})
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT downed FROM abyss_active").WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"downed"}).AddRow(false))
	mock.ExpectQuery("SELECT item_type,item_data,equip_on_bank").WithArgs(int64(31), "player").
		WillReturnRows(sqlmock.NewRows([]string{"item_type", "item_data", "equip_on_bank"}).AddRow("gear", grant, false))
	mock.ExpectRollback()
	body, err := json.Marshal(map[string]any{"id": 31, "quoted_gold": value + 1})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/abyss/loot/sell_junk", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	(&WebServer{bot: &Bot{DB: db}}).handleAbyssSellJunkLoot(rec, req, "player")
	if !strings.Contains(rec.Body.String(), "junk value changed") {
		t.Fatalf("response = %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCurrentRunLootManifestUsesStructuredGrantData(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	gear := content.Gear{
		ID: "ABYSS_BERSERKER_RING", Name: "Berserker Ring", Slot: content.SlotFinger1,
		Rarity: content.RarityEpic, MaxDurability: 70, Stats: content.Stats{STR: 120},
		Quality: 3, SetID: "predator", Corrupted: true,
	}
	grant, err := json.Marshal(abyssLootGrant{
		Type: "gear", Gear: &gear,
		SmartLoot: true, SmartLootReason: abyssSmartLootEmpty,
	})
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT id, label, depth, item_type, item_data, equip_on_bank").
		WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"id", "label", "depth", "item_type", "item_data", "equip_on_bank"}).
			AddRow(17, "<script>alert(1)</script> [s:Finger1]", 23, "gear", grant, true).
			AddRow(
				18,
				"Unidentified Finger1",
				23,
				"gear",
				mustAbyssLootGrantJSON(t, abyssLootGrant{
					Type: "gear", SetPity: true, SetPitySetID: "predator", Wishlist: true,
					Gear: &content.Gear{
						ID: "SECRET", Slot: content.SlotFinger1, Rarity: content.RarityLegendary,
						Stats: content.Stats{STR: 9_999}, Quality: 5,
						SetID: "predator", Unidentified: true,
					},
				}),
				false,
			))

	equipped := map[content.GearSlot]content.Gear{
		content.SlotFinger1: {ID: "OLD_RING", Slot: content.SlotFinger1, Stats: content.Stats{STR: 10}},
		content.SlotWaist:   {ID: "ABYSS_TITAN_BELT", Slot: content.SlotWaist, SetID: "warden"},
	}
	manifest := (&Bot{DB: db}).currentRunLootManifest("player", equipped, map[string]bool{gear.ID: true})
	if len(manifest) != 2 {
		t.Fatalf("manifest length = %d", len(manifest))
	}
	row := manifest[0]
	if row.EscrowID != 17 || row.Depth != 23 || row.Source != "Dropped floor 23" || row.GearID != gear.ID {
		t.Fatalf("manifest identity = %#v", row)
	}
	if !row.AlreadyOwned || !row.Corrupted || row.SetID != "predator" || row.SetMax != 6 || row.Quality != 3 {
		t.Fatalf("manifest metadata = %#v", row)
	}
	if !row.SmartLoot || row.SmartLootReason != abyssSmartLootEmpty || row.SmartLootLabel != "SMART · EMPTY SLOT" {
		t.Fatalf("manifest smart-loot metadata = %#v", row)
	}
	if !row.CanEquipBest || !row.EquipOnBank || row.CRDelta <= 0 {
		t.Fatalf("manifest comparison = %#v", row)
	}
	if strings.Contains(string(row.Label), "<script>") || !strings.Contains(string(row.Label), "&lt;script&gt;") {
		t.Fatalf("unsafe label = %q", row.Label)
	}
	hidden := manifest[1]
	if !hidden.Unidentified || hidden.GearID != "" || hidden.CR != 0 || hidden.Score != 0 || hidden.Quality != 0 || hidden.SetID != "" || hidden.CanEquipBest {
		t.Fatalf("unidentified gear leaked metadata: %#v", hidden)
	}
	if !hidden.SetPity || hidden.SetPityLabel != "SET PITY · 3→4" {
		t.Fatalf("unidentified set-pity reason was hidden: %#v", hidden)
	}
	if !hidden.Wishlist || hidden.WishlistLabel != "Wishlist guarantee" {
		t.Fatalf("unidentified wishlist reason was hidden: %#v", hidden)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func mustAbyssLootGrantJSON(t *testing.T, grant abyssLootGrant) []byte {
	t.Helper()
	data, err := json.Marshal(grant)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestConsumeAndEquipAbyssEscrowGearIsAtomic(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	gear := content.Gear{ID: "upgrade", Slot: content.SlotMainHand, MaxDurability: 80}
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM abyss_escrow_loot").
		WithArgs(int64(31), "player").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT gear_id, durability, item_data FROM user_gear").
		WithArgs("player", string(content.SlotMainHand)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO user_gear").
		WithArgs("player", string(content.SlotMainHand), gear.ID, gear.MaxDurability, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := (&Bot{DB: db}).consumeAndEquipAbyssEscrowGear("player", 31, gear); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssInventoryPresentationContracts(t *testing.T) {
	paths := []string{
		"web_abyss_inventory_ui.go",
		"webassets/abyss.html",
		"webassets/abyss_inventory_ui.html",
		"webassets/abyss_duplicate_guard.css",
		"webassets/abyss_run_loot_sell.css",
		"webassets/abyss_ui200.css",
		"../db/migrations/0074_abyss_loot_presentation.up.sql",
	}
	var source strings.Builder
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		source.Write(data)
	}
	for _, required := range []string{
		"data-slot-icon",
		"ab-loot-question",
		"ab-owned-tag",
		"ab-set-tag",
		"ab-loot-corrupted",
		"toggleRunLootDetail",
		"/api/abyss/loot/equip_best",
		"/api/abyss/loot/sell_junk",
		"ab-sell-junk",
		"quoted_gold",
		"lootSort",
		"already_owned",
		"equip_on_bank",
		"FOR UPDATE",
		"data-max-dur",
		"data-quality",
		"data-vendor",
		"ab-bp-attuned",
		"ab-temper-pips",
		"ab-socket-pips",
		"class=\"ab-cmp\"",
		"ab-neg",
		"ab-new",
		"Dropped floor",
		"data-smart-loot",
		"ab-smart-loot-tag",
		"data-set-pity",
		"ab-set-pity-tag",
		"lootDuplicateGuard",
		"recent_gear_protected",
		"duplicate_floor_window",
		"Recent-drop guard",
	} {
		if !strings.Contains(source.String(), required) {
			t.Errorf("inventory presentation contract missing %q", required)
		}
	}
}
