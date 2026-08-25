package bot

import (
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
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
	if abyssActiveInsanityCosmetic(before) == abyssActiveInsanityCosmetic(before.AddDate(0, 0, 1)) {
		t.Error("Insanity cosmetic rotation did not advance on the next UTC day")
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
		"/api/abyss/shop/scratch", "/api/abyss/shop/gift_create", "/api/abyss/shop/gift_redeem",
		"/api/abyss/economy/loan", "/api/abyss/economy/tax_rebate", "/api/ah/watch", "/api/ah/notices",
		"/api/ah/bulk_relist", "/api/ah/material_order", "/api/ah/material_fill", "/api/ah/material_cancel", "/api/ah/bid",
		"Cheapest active Legendary", "Insanity", "seller receives the exact buy-now price", "Attuned items are soulbound",
		"Most Taxed This Season", "Relist all expired", "Daily scratch card", "Gold → token bundle",
		"role=\"tablist\" aria-label=\"Token Shop category\"", "role=\"tablist\" aria-label=\"Auction history\"",
		"Redemption fee 0g / 0 tokens", "winning bid is the exact total", "Cancellation fee 0g",
	} {
		if !strings.Contains(combined, required) {
			t.Errorf("economy contract is missing %q", required)
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
		WithArgs("bidder", "Bid refunded: Broken Blade could not be delivered safely.", int64(500)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	(&Bot{DB: database}).settleAbyssAuctionBid("broken-listing")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
