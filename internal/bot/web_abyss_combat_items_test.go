package bot

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestAbyssPostHocOddsUseCompletedFloorRisk(t *testing.T) {
	t.Parallel()

	tier, _ := abyssTierByKey("normal")
	risk := abyssRiskPct(12, tier, 2_500)
	if got := abyssPostHocSurvivalChance(12, tier, 2_500); got != 100-risk {
		t.Fatalf("survival chance = %d, want %d", got, 100-risk)
	}
}

func TestAbyssCorruptedConsumableBacklash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id   string
		hp   int
		want int
	}{
		{"corrupted_great_health_potion", 1_000, 100},
		{"corrupted_rejuvenation_potion", 1_000, 100},
		{"corrupted_strength_elixir", 1_000, 50},
		{"great_health_potion", 1_000, 0},
		{"corrupted_strength_elixir", 1, 1},
	}
	for _, test := range tests {
		if got := corruptedConsumableBacklash(test.id, test.hp); got != test.want {
			t.Errorf("backlash(%q, %d) = %d, want %d", test.id, test.hp, got, test.want)
		}
	}
}

func TestAbyssDefensiveRunesStackByElement(t *testing.T) {
	t.Parallel()

	equipped := map[content.GearSlot]content.Gear{
		content.SlotChest: {Rune: content.DefensiveRuneName(content.ElementFire)},
		content.SlotHead:  {Rune: content.DefensiveRuneName(content.ElementFire)},
		content.SlotFeet:  {Rune: content.DefensiveRuneName(content.ElementWater)},
	}
	if got := defensiveRuneResistPct(equipped, content.ElementFire); got != 10 {
		t.Fatalf("fire resistance = %d, want 10", got)
	}
	if got := defensiveRuneResistPct(equipped, content.ElementWater); got != 5 {
		t.Fatalf("water resistance = %d, want 5", got)
	}
	if got := defensiveRuneResistPct(equipped, content.ElementAir); got != 0 {
		t.Fatalf("air resistance = %d, want 0", got)
	}
}

func TestAbyssSentimentalValueBonus(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC)
	gear := content.Gear{
		Name:    "Old Friend",
		FoundAt: now.Add(-31 * 24 * time.Hour).Format(time.RFC3339),
		Stats:   content.Stats{HP: 1_000, STR: 250, DEF: 99, STN: -100},
	}
	bonus := sentimentalValueBonus(gear, now)
	if bonus.HP != 10 || bonus.STR != 2 || bonus.DEF != 0 || bonus.STN != 0 {
		t.Fatalf("sentimental bonus = %+v", bonus)
	}
	gear.FoundAt = now.Add(-29 * 24 * time.Hour).Format(time.RFC3339)
	if got := sentimentalValueBonus(gear, now); got != (content.Stats{}) {
		t.Fatalf("young item bonus = %+v, want zero", got)
	}
}

func TestAbyssLootVariantsAndForecast(t *testing.T) {
	t.Parallel()

	if got := abyssBeamClass(content.RarityEternal, false); got != "beam-eternal" {
		t.Fatalf("eternal beam = %q", got)
	}
	if got := abyssBeamClass(content.RarityCommon, true); got != "beam-doomed" {
		t.Fatalf("doomed beam = %q", got)
	}
	forecast := abyssDropForecastData(1, 1)
	total := forecast.Ultimate + forecast.Title + forecast.Unique + forecast.Artifact + forecast.Enchant + forecast.Skill + forecast.Consumable + forecast.Gear + forecast.Common
	if total < 0.999999 || total > 1.000001 || forecast.Common < 0 {
		t.Fatalf("invalid loot forecast: total=%f common=%f", total, forecast.Common)
	}
	forecast = abyssDropForecastData(100, 100)
	total = forecast.Ultimate + forecast.Title + forecast.Unique + forecast.Artifact + forecast.Enchant + forecast.Skill + forecast.Consumable + forecast.Gear + forecast.Common
	if total < 0.999999 || total > 1.000001 || forecast.Common < 0 {
		t.Fatalf("invalid saturated loot forecast: total=%f common=%f", total, forecast.Common)
	}
	lucid := lucidInsanityVariant(content.Gear{Name: "Insanity", Stats: content.Stats{HP: 1_000, STR: 100, DEF: -100}})
	if !lucid.Lucid || lucid.Name != "Lucid Insanity" || lucid.Stats.HP != 800 || lucid.Stats.STR != 80 || lucid.Stats.DEF != 0 {
		t.Fatalf("lucid variant = %+v", lucid)
	}
}

func TestAbyssConsumableEscrowGrantStacksToCap(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectExec("INSERT INTO user_consumables").
		WithArgs("user1", "rejuvenation_potion", 9, abyssConsumableStackCap).
		WillReturnResult(sqlmock.NewResult(1, 1))
	bot := &Bot{DB: db}
	if err := bot.applyAbyssLootGrant("user1", abyssLootGrant{Type: "cons", ConsID: "rejuvenation_potion", ConsDur: 9}); err != nil {
		t.Fatalf("apply stacked grant: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}

	mock.ExpectExec("INSERT INTO user_consumables").
		WithArgs("user1", "rejuvenation_potion", 1, abyssConsumableStackCap).
		WillReturnError(errors.New("write failed"))
	if err := bot.applyAbyssLootGrant("user1", abyssLootGrant{Type: "cons", ConsID: "rejuvenation_potion"}); err == nil {
		t.Fatal("failed stacked grant reported success")
	}
}

func TestAbyssSetTradeSelectsMissingItemAndOnlySpares(t *testing.T) {
	t.Parallel()

	catalog := content.AbyssSetCatalog("harvester")
	if len(catalog) != 3 {
		t.Fatalf("harvester catalog has %d items, want 3", len(catalog))
	}
	owned := map[string]bool{catalog[0].ID: true, catalog[1].ID: true}
	offer, ok := abyssRotatingMissingSetItem(catalog, owned, 2026, 35)
	if !ok || offer.ID != catalog[2].ID {
		t.Fatalf("missing offer = %q, %t; want %q", offer.ID, ok, catalog[2].ID)
	}
	items := []abyssSetTradeItem{
		{invID: 1, gear: catalog[0]}, {invID: 2, gear: catalog[0]},
		{invID: 3, gear: catalog[1]}, {invID: 4, gear: catalog[1]},
		{invID: 5, gear: catalog[1]},
	}
	spares := abyssSetTradeSpareIDs(items, "harvester")
	if len(spares) != 3 || spares[0] != 2 || spares[1] != 4 || spares[2] != 5 {
		t.Fatalf("spare IDs = %v, want [2 4 5]", spares)
	}
	spares = abyssSetTradeSpareIDs([]abyssSetTradeItem{{invID: 6, gear: catalog[0]}, {invID: 7, gear: catalog[1]}}, "harvester", map[string]bool{catalog[0].ID: true, catalog[1].ID: true})
	if len(spares) != 2 || spares[0] != 6 || spares[1] != 7 {
		t.Fatalf("equipped-backed spare IDs = %v, want [6 7]", spares)
	}
}

func TestAbyssSetTradeTransaction(t *testing.T) {
	catalog := content.AbyssSetCatalog("harvester")
	if len(catalog) != 3 {
		t.Fatalf("harvester catalog has %d items, want 3", len(catalog))
	}
	for _, test := range []struct {
		name      string
		insertErr error
		wantOK    bool
	}{
		{name: "commit", wantOK: true},
		{name: "rollback failed grant", insertErr: errors.New("write failed")},
	} {
		t.Run(test.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = db.Close() }()

			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta("SELECT id, gear_id, item_data FROM user_inventory WHERE client_uid=$1 ORDER BY id FOR UPDATE")).
				WithArgs("user1").
				WillReturnRows(sqlmock.NewRows([]string{"id", "gear_id", "item_data"}).
					AddRow(1, catalog[0].ID, nil).
					AddRow(2, catalog[0].ID, nil).
					AddRow(3, catalog[1].ID, nil).
					AddRow(4, catalog[1].ID, nil))
			mock.ExpectQuery(regexp.QuoteMeta("SELECT gear_id, item_data FROM user_gear WHERE client_uid=$1 FOR UPDATE")).
				WithArgs("user1").
				WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}))
			mock.ExpectExec("DELETE FROM user_inventory").WithArgs(int64(2), "user1").WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectExec("DELETE FROM user_inventory").WithArgs(int64(4), "user1").WillReturnResult(sqlmock.NewResult(0, 1))
			insert := mock.ExpectExec("INSERT INTO user_inventory").
				WithArgs("user1", catalog[2].ID, catalog[2].MaxDurability, sqlmock.AnyArg())
			if test.insertErr != nil {
				insert.WillReturnError(test.insertErr)
				mock.ExpectRollback()
			} else {
				insert.WillReturnResult(sqlmock.NewResult(5, 1))
				mock.ExpectCommit()
			}

			server := &WebServer{bot: &Bot{DB: db}}
			request := httptest.NewRequest(http.MethodPost, "/api/abyss/set_trade", strings.NewReader(`{"set_id":"harvester"}`))
			response := httptest.NewRecorder()
			server.handleAbyssSetTrade(response, request, "user1")
			if gotOK := strings.Contains(response.Body.String(), `"ok":true`); gotOK != test.wantOK {
				t.Fatalf("response = %s, want ok=%t", response.Body.String(), test.wantOK)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAbyssMindControlCaptureFlipsAtOneHP(t *testing.T) {
	t.Parallel()

	mob := &content.Mob{Stats: content.Stats{HP: 7}, CurrentHP: 7, MaxHP: 100}
	abyssMindControlCapture(mob)
	if mob.Stats.HP != 1 || mob.CurrentHP != 1 || mob.Loyalty != 7 {
		t.Fatalf("captured mob = HP %d/%d, loyalty %d", mob.Stats.HP, mob.CurrentHP, mob.Loyalty)
	}
	if !abyssPetNervous(mob.Loyalty) || abyssPetNervous(10) || abyssPetNervous(0) {
		t.Fatal("pet nervous threshold is not the strict 1-9% loyalty band")
	}
}

func TestAbyssPerfectPunchAndBuildSort(t *testing.T) {
	t.Parallel()

	if sockets, perfect := abyssPunchSocketResult(3, 0.09); sockets != 5 || !perfect {
		t.Fatalf("perfect punch = %d, %t", sockets, perfect)
	}
	if sockets, perfect := abyssPunchSocketResult(3, 0.10); sockets != 4 || perfect {
		t.Fatalf("boundary punch = %d, %t", sockets, perfect)
	}
	gear := content.Gear{Stats: content.Stats{HP: 300, STR: 40, DEF: 70, INT: 90}}
	if got := abyssLootMainStat(gear, abyssBuildKits["arcanist"]); got != 90 {
		t.Fatalf("arcanist main stat = %d", got)
	}
	if got := abyssLootMainStat(gear, abyssBuildKits["survival"]); got != 370 {
		t.Fatalf("survival main stat = %d", got)
	}
}

func TestAbyssCombatAndItemPresentationContracts(t *testing.T) {
	t.Parallel()

	source := ""
	for _, name := range []string{
		"webassets/abyss.html", "webassets/abyss_live.html", "webassets/abyss_pixel.html",
		"webassets/abyss_live.css", "webassets/abyss_loot_presentation.css",
	} {
		data, err := webAssets.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		source += string(data)
	}
	for _, required := range []string{
		"survival_chance_pct", "Post-fight odds", "ally-converted",
		"!unit.is_player", "Upgrades first", "Build stat", "ab-charm-dangle",
		"Set Trading Post", "Convert 3rd Leg+", "/api/abyss/set_trade",
		"beam-eternal", "beam-doomed", "ab-loot-foil",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("Abyss presentation is missing %q", required)
		}
	}
}
