package bot

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"ts3news/internal/content"
)

func TestAbyssEconomyMechanics(t *testing.T) {
	t.Parallel()

	if got := abyssDiscountedCost(9); got != 5 {
		t.Errorf("40%% happy-accident price = %d, want 5", got)
	}
	if abyssHappyAccidentIndex(time.Date(2026, 8, 25, 23, 59, 0, 0, time.FixedZone("late", 9*3600)), 7) !=
		abyssHappyAccidentIndex(time.Date(2026, 8, 25, 0, 1, 0, 0, time.UTC), 7) {
		t.Error("happy-accident selection is not UTC-day stable")
	}
	if abyssHappyAccidentIndex(time.Now(), 0) != -1 {
		t.Error("empty deal catalog must return no selection")
	}
	dealCount := 0
	for _, item := range abyssShopCatalog {
		cost, deal := abyssShopEffectiveCost(item, time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC))
		if deal {
			dealCount++
			if cost != abyssDiscountedCost(item.Cost) {
				t.Errorf("daily deal cost = %d, want %d", cost, abyssDiscountedCost(item.Cost))
			}
		}
	}
	if dealCount != 1 {
		t.Errorf("daily happy accidents = %d, want exactly one", dealCount)
	}
	if abyssTokenBundleRate(0) != 100_000 || abyssTokenBundleRate(1) != 120_000 || abyssTokenBundleRate(-2) != 100_000 {
		t.Error("sliding token bundle rates are incorrect")
	}
	if abyssVendorLoyaltyPercent(4) != 0 || abyssVendorLoyaltyPercent(5) != 2 {
		t.Error("vendor loyalty must unlock after five completed same-item sales")
	}
	if abyssLoanLimit(100, 0) != 50 || abyssLoanLimit(70, 30) != 20 || abyssLoanLimit(50, 50) != 0 {
		t.Error("cache loan limit exceeded the original-cache 50% ceiling")
	}
	if abyssLoanFee(0) != 0 || abyssLoanFee(1) != 1 || abyssLoanFee(101) != 11 {
		t.Error("cache loan fee is not a ceiling-rounded 10%")
	}
	if abyssJackpotHelperShare(-1) != 0 || abyssJackpotHelperShare(9) != 0 || abyssJackpotHelperShare(1_000) != 100 {
		t.Error("jackpot helper split is not exactly 10%")
	}
	if abyssRepairSubscriptionCharge(12_000, true) != 0 || abyssRepairSubscriptionCharge(12_000, false) != 12_000 {
		t.Error("repair subscription does not cover the full repair charge")
	}
}

func TestAbyssScratchPostedOddsBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		roll float64
		want int
	}{{0, 1_000}, {0.009999, 1_000}, {0.01, 150}, {0.099999, 150}, {0.10, 75}, {0.299999, 75}, {0.30, 0}, {0.999, 0}}
	for _, test := range tests {
		if got := abyssScratchReward(test.roll); got != test.want {
			t.Errorf("scratch reward at %.6f = %d, want %d", test.roll, got, test.want)
		}
	}
}

func TestAbyssEconomyCalendarAndInsurance(t *testing.T) {
	t.Parallel()

	before := time.Date(2026, 8, 23, 23, 59, 0, 0, time.UTC)
	after := before.Add(2 * time.Minute)
	if abyssEconomyWeek(before) == abyssEconomyWeek(after) {
		t.Error("ISO week key did not roll over at the Monday boundary")
	}
	friday := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	if abyssWeeklyInsanityCosmetic(friday) != abyssWeeklyInsanityCosmetic(friday.AddDate(0, 0, 2)) {
		t.Error("Insanity cosmetic rotation changed within one ISO week")
	}
	if abyssWeeklyInsanityCosmetic(friday) == abyssWeeklyInsanityCosmetic(friday.AddDate(0, 0, 3)) {
		t.Error("Insanity cosmetic rotation did not advance at the next ISO week")
	}
	if reset := abyssWeeklyCosmeticReset(friday); !reset.Equal(time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("weekly cosmetic reset = %v, want Monday 00:00 UTC", reset)
	}
	for _, test := range []struct {
		yesterday, twoDays, available bool
		continues, uses               bool
	}{{true, false, false, true, false}, {false, true, true, true, true}, {false, true, false, false, false}, {false, false, true, false, false}} {
		continues, uses := abyssBountyContinues(test.yesterday, test.twoDays, test.available)
		if continues != test.continues || uses != test.uses {
			t.Errorf("insurance (%t,%t,%t) = (%t,%t), want (%t,%t)", test.yesterday, test.twoDays, test.available, continues, uses, test.continues, test.uses)
		}
	}
}

func TestAbyssAntiSnipeExactWindow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		expiry time.Time
		want   time.Time
	}{{now.Add(30 * time.Second), now.Add(90 * time.Second)}, {now.Add(time.Minute), now.Add(2 * time.Minute)}, {now.Add(time.Minute + time.Nanosecond), now.Add(time.Minute + time.Nanosecond)}, {now, now}} {
		if got := abyssAntiSnipeExpiry(now, test.expiry); !got.Equal(test.want) {
			t.Errorf("anti-snipe expiry = %v, want %v", got, test.want)
		}
	}
}

func TestAbyssMaterialOrderTotalRejectsOverflow(t *testing.T) {
	t.Parallel()

	if total, ok := abyssMaterialOrderTotal(math.MaxInt64, 2); ok || total != 0 {
		t.Fatalf("overflowing order total = %d, %t; want rejected", total, ok)
	}
	if total, ok := abyssMaterialOrderTotal(125, 8); !ok || total != 1_000 {
		t.Fatalf("valid order total = %d, %t; want 1000, true", total, ok)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/ah/material_order",
		strings.NewReader(`{"material":"dust","count":2,"unit_price":9223372036854775807}`))
	response := httptest.NewRecorder()
	(&WebServer{bot: &Bot{}}).handleAHMaterialOrder(response, request, "buyer")
	if body := response.Body.String(); !strings.Contains(body, `"ok":false`) || !strings.Contains(body, "order total is too large") {
		t.Fatalf("overflow response = %s", body)
	}
}

func TestAbyssEconomyPlayerControlsAndRoutes(t *testing.T) {
	t.Parallel()

	abyssPage, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	ahPage, err := webAssets.ReadFile("webassets/ah.html")
	if err != nil {
		t.Fatal(err)
	}
	routes, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	ahServer, err := os.ReadFile("web_ah.go")
	if err != nil {
		t.Fatal(err)
	}
	combined := string(abyssPage) + string(ahPage) + string(routes) + string(ahServer)
	for _, required := range []string{
		"/api/abyss/shop/token_bundle", "/api/abyss/shop/potion_subscription", "/api/abyss/shop/repair_subscription",
		"/api/abyss/shop/auto_insure", "Auto-insure:",
		"Depth pressure: 200g × (1 + floor(depth/10)²)", "Legendary / Mythic / Divine / Celestial / Eternal",
		"Current full-repair total: {{gold .RepairAllCost}}",
		"/api/abyss/shop/scratch", "/api/abyss/shop/gift_create", "/api/abyss/shop/gift_redeem",
		"/api/abyss/economy/loan", "/api/abyss/economy/tax_rebate", "/api/ah/watch", "/api/ah/notices",
		"/api/ah/bulk_relist", "/api/ah/material_order", "/api/ah/material_fill", "/api/ah/material_cancel", "/api/ah/bid",
		"Cheapest active Legendary", "Insanity", "seller community tax", "Attuned items are soulbound",
		"Most Taxed This Season", "Relist all expired", "Daily scratch card", "Gold → token bundle",
		"role=\"tablist\" aria-label=\"Token Shop category\"", "role=\"tablist\" aria-label=\"Auction history\"",
		"Weekly cosmetic", "refreshes {{.RotationEnds}}",
		"Redemption fee 0g / 0 tokens", "winning bid is the exact total", "Cancellation fee 0g",
		"Unavailable until identified", "Identify first", "Unlock in Abyss", "No sold listings yet.",
		"Your leading bid", "Exact proceeds:", "Confirm material sale", "Exact end: {{.Expires}}",
	} {
		if !strings.Contains(combined, required) {
			t.Errorf("economy contract is missing %q", required)
		}
	}
	for _, handler := range []string{
		"handleAHBuyAPI", "handleAHListAPI", "handleAHWatch", "handleAHBulkRelist",
		"handleAHMaterialOrder", "handleAHMaterialFill", "handleAHMaterialCancel", "handleAHBid",
	} {
		if !strings.Contains(string(routes), "guardAbyssCoreAction(s."+handler+")") {
			t.Errorf("auction mutation %s is missing idempotency and replay protection", handler)
		}
	}
}

func TestAbyssEconomyMigrationContract(t *testing.T) {
	t.Parallel()

	migration, err := os.ReadFile("../db/migrations/0080_abyss_economy.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	source := string(migration)
	for _, required := range []string{
		"abyss_economy_profiles", "abyss_ah_watchlist", "abyss_economy_events", "abyss_material_orders",
		"abyss_shop_gifts", "abyss_shop_cosmetics", "abyss_vendor_sales", "economy_loan_fee",
		"economy_loan_principal", "current_bid", "bidder_uid", "idx_auction_house_bid_settlement",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("economy migration is missing %q", required)
		}
	}
}

func TestAbyssAutoInsurePreferencePersists(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	uid := "insured-player"
	mock.ExpectExec("INSERT INTO abyss_economy_profiles \\(client_uid,auto_insure\\)").
		WithArgs(uid, true).WillReturnResult(sqlmock.NewResult(1, 1))
	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/shop/auto_insure", strings.NewReader(`{"enabled":true}`))
	response := httptest.NewRecorder()
	server.handleAbyssAutoInsure(response, request, uid)
	if body := response.Body.String(); !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"enabled":true`) {
		t.Fatalf("preference response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyAbyssAutoInsuranceChargesOrUsesVoucherInCallerTransaction(t *testing.T) {
	for _, test := range []struct {
		name      string
		gold      int64
		voucher   string
		wantCost  int64
		wantFree  bool
		wantErr   error
		wantDebit bool
	}{
		{name: "premium", gold: 500, voucher: "0", wantCost: 125, wantDebit: true},
		{name: "voucher", gold: 0, voucher: "1", wantFree: true},
		{name: "unaffordable", gold: 124, voucher: "0", wantErr: errAbyssAutoInsuranceFunds},
	} {
		t.Run(test.name, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			mock.ExpectBegin()
			tx, err := database.Begin()
			if err != nil {
				t.Fatal(err)
			}
			uid := "auto-insured-player"
			mock.ExpectQuery("SELECT gold FROM users").WithArgs(uid).
				WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(test.gold))
			key := abyssFreeInsuranceKey(uid)
			mock.ExpectExec("INSERT INTO app_meta").WithArgs(key).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery("SELECT value FROM app_meta").WithArgs(key).
				WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(test.voucher))
			if test.wantFree {
				mock.ExpectExec("UPDATE app_meta SET value='0'").WithArgs(key).WillReturnResult(sqlmock.NewResult(0, 1))
			}
			if test.wantDebit {
				mock.ExpectExec("UPDATE users SET gold=gold-\\$1").WithArgs(int64(125), uid).
					WillReturnResult(sqlmock.NewResult(0, 1))
			}
			cost, free, gotErr := applyAbyssAutoInsurance(tx, uid, abyssAutoInsurancePlan{Applied: true, Percent: 25, Cost: 125})
			if cost != test.wantCost || free != test.wantFree || !errors.Is(gotErr, test.wantErr) {
				t.Fatalf("apply = (%d, %t, %v), want (%d, %t, %v)", cost, free, gotErr, test.wantCost, test.wantFree, test.wantErr)
			}
			if test.wantErr != nil {
				mock.ExpectRollback()
				_ = tx.Rollback()
			} else {
				mock.ExpectCommit()
				if err := tx.Commit(); err != nil {
					t.Fatal(err)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAbyssAutoInsureMigrationContract(t *testing.T) {
	t.Parallel()

	up, err := os.ReadFile("../db/migrations/0094_abyss_auto_insure.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../db/migrations/0094_abyss_auto_insure.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(up), "auto_insure BOOLEAN NOT NULL DEFAULT FALSE") {
		t.Fatal("auto-insure migration does not add a safe opt-in default")
	}
	if !strings.Contains(string(down), "DROP COLUMN IF EXISTS auto_insure") {
		t.Fatal("auto-insure rollback does not remove the preference")
	}
}

func TestAbyssEconomyLoanCommitsGoldAndFeeAtomically(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	uid := "loan-player"
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT escrow,economy_loan_principal,downed FROM abyss_active").
		WithArgs(uid).WillReturnRows(sqlmock.NewRows([]string{"escrow", "principal", "downed"}).AddRow(1000, 0, false))
	mock.ExpectExec("UPDATE abyss_active SET escrow=escrow-\\$1").
		WithArgs(int64(400), int64(40), uid).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET gold=gold+$1 WHERE client_uid=$2")).
		WithArgs(int64(400), uid).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT gold FROM users WHERE client_uid=$1")).
		WithArgs(uid).WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(1_400))

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/economy/loan", strings.NewReader(`{"amount":400}`))
	response := httptest.NewRecorder()
	server.handleAbyssEconomyLoan(response, request, uid)
	if body := response.Body.String(); !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"fee_due":40`) {
		t.Fatalf("loan response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssAuctionBidReservesFundsAndExtendsFinalMinute(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	uid := "bidder"
	expires := time.Now().Add(30 * time.Second)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT seller_uid,item_type,item_id,item_data,price,current_bid,bidder_uid,expires_at").
		WithArgs("listing-1").WillReturnRows(sqlmock.NewRows([]string{"seller_uid", "item_type", "item_id", "item_data", "price", "current_bid", "bidder_uid", "expires_at"}).
		AddRow("seller", "gear", "G1", []byte(`{"ID":"G1"}`), 1000, 0, nil, expires))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET gold=gold-$1 WHERE client_uid=$2 AND gold >= $1")).
		WithArgs(int64(600), uid).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE auction_house SET current_bid=\\$1").
		WithArgs(int64(600), uid, sqlmock.AnyArg(), "listing-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT gold FROM users WHERE client_uid=$1")).
		WithArgs(uid).WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(400))

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/ah/bid", strings.NewReader(`{"id":"listing-1","amount":600}`))
	response := httptest.NewRecorder()
	server.handleAHBid(response, request, uid)
	if body := response.Body.String(); !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"extended":true`) {
		t.Fatalf("bid response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssMaterialOrderCancelRefundsEscrowAtomically(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT escrow_gold FROM abyss_material_orders").
		WithArgs(int64(17), "buyer").
		WillReturnRows(sqlmock.NewRows([]string{"escrow_gold"}).AddRow(int64(420)))
	mock.ExpectExec("UPDATE abyss_material_orders SET remaining=0").
		WithArgs(int64(17)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET gold=gold+$1 WHERE client_uid=$2")).
		WithArgs(int64(420), "buyer").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT gold FROM users WHERE client_uid=$1")).
		WithArgs("buyer").WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(int64(9_999)))

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/ah/material_cancel", strings.NewReader(`{"id":17}`))
	response := httptest.NewRecorder()
	server.handleAHMaterialCancel(response, request, "buyer")
	if body := response.Body.String(); !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"refund":420`) {
		t.Fatalf("cancel response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAHBuyRejectsCorruptPayloadBeforeMovingGold(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT item_type, item_id, item_name, item_data").
		WithArgs("listing-1").
		WillReturnRows(sqlmock.NewRows([]string{"item_type", "item_id", "item_name", "item_data", "price", "seller_uid", "durability", "current_bid", "bidder_uid"}).
			AddRow("gear", "G1", "Broken Blade", []byte("{"), int64(100), "seller", nil, int64(0), nil))
	mock.ExpectRollback()

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/ah/buy", strings.NewReader(`{"id":"listing-1"}`))
	response := httptest.NewRecorder()
	server.handleAHBuyAPI(response, request, "buyer")
	if body := response.Body.String(); !strings.Contains(body, `"ok":false`) || !strings.Contains(body, "listing has invalid gear data") {
		t.Fatalf("corrupt listing response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAHEnrichListingRedactsUnidentifiedGear(t *testing.T) {
	gear := content.Gear{
		ID:           "SECRET_HEAD",
		Name:         "Crown of the Last Star",
		Slot:         content.SlotHead,
		Rarity:       content.RarityCelestial,
		Stats:        content.Stats{STR: 999, DEF: 777},
		Unidentified: true,
	}
	payload, err := json.Marshal(gear)
	if err != nil {
		t.Fatal(err)
	}
	listing := ahListingView{ItemType: "gear", ItemID: gear.ID, Name: gear.Name}
	ahEnrichListing(&listing, payload)

	if listing.Name != "Unidentified Head" || listing.Rarity != "Unknown" || listing.GS != 0 || listing.InspectJSON != "" {
		t.Fatalf("unidentified listing leaked metadata: %+v", listing)
	}
}

func TestAHListRejectsUnidentifiedGearBeforeRemoval(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	gear := content.Gear{ID: "SECRET_HEAD", Name: "Crown of the Last Star", Slot: content.SlotHead, Rarity: content.RarityCelestial, Unidentified: true}
	payload, err := json.Marshal(gear)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, durability, item_data FROM user_inventory").
		WithArgs(int64(17), "seller").
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "durability", "item_data"}).AddRow(gear.ID, 80, string(payload)))
	mock.ExpectRollback()

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/ah/list", strings.NewReader(`{"inv_id":17}`))
	response := httptest.NewRecorder()
	server.handleAHListAPI(response, request, "seller")
	if body := response.Body.String(); !strings.Contains(body, `"ok":false`) || !strings.Contains(body, "identify the item before listing it") {
		t.Fatalf("unidentified list response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAHBuyRejectsUnidentifiedGearBeforeMovingGold(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	gear := content.Gear{ID: "SECRET_HEAD", Name: "Crown of the Last Star", Slot: content.SlotHead, Rarity: content.RarityCelestial, Unidentified: true}
	payload, err := json.Marshal(gear)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT item_type, item_id, item_name, item_data").
		WithArgs("listing-secret").
		WillReturnRows(sqlmock.NewRows([]string{"item_type", "item_id", "item_name", "item_data", "price", "seller_uid", "durability", "current_bid", "bidder_uid"}).
			AddRow("gear", gear.ID, gear.Name, payload, int64(999_999), "seller", 80, int64(0), nil))
	mock.ExpectRollback()

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/ah/buy", strings.NewReader(`{"id":"listing-secret"}`))
	response := httptest.NewRecorder()
	server.handleAHBuyAPI(response, request, "buyer")
	if body := response.Body.String(); !strings.Contains(body, `"ok":false`) || !strings.Contains(body, "listing unavailable") {
		t.Fatalf("unidentified buy response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAHBidRejectsUnidentifiedGearBeforeReservingGold(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	gear := content.Gear{ID: "SECRET_HEAD", Name: "Crown of the Last Star", Slot: content.SlotHead, Rarity: content.RarityCelestial, Unidentified: true}
	payload, err := json.Marshal(gear)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT seller_uid,item_type,item_id,item_data,price,current_bid,bidder_uid,expires_at").
		WithArgs("listing-secret").
		WillReturnRows(sqlmock.NewRows([]string{"seller_uid", "item_type", "item_id", "item_data", "price", "current_bid", "bidder_uid", "expires_at"}).
			AddRow("seller", "gear", gear.ID, payload, int64(999_999), int64(0), nil, time.Now().Add(time.Hour)))
	mock.ExpectRollback()

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/ah/bid", strings.NewReader(`{"id":"listing-secret","amount":500000}`))
	response := httptest.NewRecorder()
	server.handleAHBid(response, request, "buyer")
	if body := response.Body.String(); !strings.Contains(body, `"ok":false`) || !strings.Contains(body, "listing unavailable") {
		t.Fatalf("unidentified bid response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAHExpiryLabelsShowRelativeAndExactTerms(t *testing.T) {
	now := time.Date(2026, time.August, 31, 12, 0, 0, 0, time.UTC)
	exact, relative := ahExpiryLabels(now.Add(26*time.Hour+15*time.Minute), now)
	if exact != "01 Sep 2026 · 14:15 UTC" || relative != "in 1d 2h" {
		t.Fatalf("expiry labels = %q, %q", exact, relative)
	}
}

func TestInventoryEquipRejectsUnidentifiedGear(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	gear := content.Gear{ID: "SECRET_HEAD", Name: "Crown of the Last Star", Slot: content.SlotHead, Unidentified: true}
	payload, err := json.Marshal(gear)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT gear_id, durability, item_data FROM user_inventory").
		WithArgs(int64(17), "delver").
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "durability", "item_data"}).AddRow(gear.ID, 80, string(payload)))

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/inventory/equip", strings.NewReader(`{"inv_id":17}`))
	response := httptest.NewRecorder()
	server.handleEquipAPI(response, request, "delver")
	if body := response.Body.String(); !strings.Contains(body, `"ok":false`) || !strings.Contains(body, "identify the item before equipping it") {
		t.Fatalf("unidentified equip response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestInventoryVendorRejectsUnidentifiedGearBeforePricing(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	gear := content.Gear{ID: "SECRET_HEAD", Name: "Crown of the Last Star", Slot: content.SlotHead, Rarity: content.RarityCelestial, Stats: content.Stats{STR: 999}, Unidentified: true}
	payload, err := json.Marshal(gear)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id,durability,item_data,acquired_at FROM user_inventory").
		WithArgs(int64(17), "delver").
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "durability", "item_data", "acquired_at"}).AddRow(gear.ID, 80, string(payload), time.Now()))
	mock.ExpectRollback()

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/inventory/sell", strings.NewReader(`{"inv_id":17}`))
	response := httptest.NewRecorder()
	server.handleSellAPI(response, request, "delver")
	if body := response.Body.String(); !strings.Contains(body, `"ok":false`) || !strings.Contains(body, "identify the item before selling it") {
		t.Fatalf("unidentified vendor response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssOwnedCosmeticReequipsWithoutSecondCharge(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO abyss_shop_cosmetics").
		WithArgs("collector", "insanity_void_aura").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("UPDATE users SET title=\\$2").
		WithArgs("collector", "Void-Touched").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT abyss_tokens FROM users").
		WithArgs("collector").WillReturnRows(sqlmock.NewRows([]string{"abyss_tokens"}).AddRow(int64(77)))

	server := &WebServer{bot: &Bot{DB: database}}
	item, ok := abyssShopByKey("insanity_void_aura")
	if !ok {
		t.Fatal("missing cosmetic fixture")
	}
	response := httptest.NewRecorder()
	server.buyAbyssShopCosmetic(response, "collector", item, item.Cost)
	if body := response.Body.String(); !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"newly_owned":false`) {
		t.Fatalf("re-equip response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssInvalidWinningBidRefundsReservedGold(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT seller_uid,bidder_uid,item_type,item_id,item_name,item_data,current_bid,durability").
		WithArgs("broken-listing").
		WillReturnRows(sqlmock.NewRows([]string{"seller_uid", "bidder_uid", "item_type", "item_id", "item_name", "item_data", "current_bid", "durability"}).
			AddRow("seller", "bidder", "gear", "G1", "Broken Blade", []byte("{"), int64(500), nil))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET gold=gold+$1 WHERE client_uid=$2")).
		WithArgs(int64(500), "bidder").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE auction_house SET current_bid=0,bidder_uid=NULL").
		WithArgs("broken-listing").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO abyss_economy_events").
		WithArgs("bidder", "Bid refunded: a hidden auction item could not be delivered safely.", int64(500)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	(&Bot{DB: database}).settleAbyssAuctionBid("broken-listing")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssWinningBidPersistsFinalSalePrice(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	gear := content.Gear{ID: "G1", Name: "Honest Blade", MaxDurability: 80}
	payload, err := json.Marshal(gear)
	if err != nil {
		t.Fatal(err)
	}
	const bid = int64(625)
	const salesTax = int64(32)
	const sellerNet = bid - salesTax

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT seller_uid,bidder_uid,item_type,item_id,item_name,item_data,current_bid,durability").
		WithArgs("won-listing").
		WillReturnRows(sqlmock.NewRows([]string{"seller_uid", "bidder_uid", "item_type", "item_id", "item_name", "item_data", "current_bid", "durability"}).
			AddRow("seller", "bidder", "gear", gear.ID, gear.Name, payload, bid, nil))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO user_inventory (client_uid,gear_id,durability,item_data) VALUES ($1,$2,$3,$4)")).
		WithArgs("bidder", gear.ID, gear.MaxDurability, payload).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET gold=gold+$1 WHERE client_uid=$2")).
		WithArgs(sellerNet, "seller").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE arcade_jackpots SET amount=amount\+\$1`).
		WithArgs(salesTax).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE auction_house SET buyer_uid=$1,sold_at=NOW(),price=$2,current_bid=0,bidder_uid=NULL WHERE id=$3")).
		WithArgs("bidder", bid, "won-listing").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO abyss_economy_events").
		WithArgs("seller", "bidder",
			"Auction sold by bid: Honest Blade · 625g gross · 32g community tax · 593g net.", sellerNet,
			"Winning bid delivered: Honest Blade · 625g reserved.", -bid).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	(&Bot{DB: database}).settleAbyssAuctionBid("won-listing")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
