package bot

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssShopLoyaltyTenthPurchaseIsFreeAndBounded(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO app_meta").WithArgs(abyssShopLoyaltyKey("delver")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT value FROM app_meta.*FOR UPDATE").WithArgs(abyssShopLoyaltyKey("delver")).WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("9"))
	mock.ExpectExec("INSERT INTO app_meta").WithArgs(abyssShopLoyaltyKey("delver"), "0").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	charged, punches, free, err := applyAbyssShopLoyalty(tx, "delver", 30)
	if err != nil || charged != 0 || punches != 0 || !free {
		t.Fatalf("loyalty result = charged %d punches %d free %v err %v", charged, punches, free, err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssShopBundlePurchaseIsAtomic(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO app_meta").WithArgs(abyssShopLoyaltyKey("delver")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT value FROM app_meta.*FOR UPDATE").WithArgs(abyssShopLoyaltyKey("delver")).WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("0"))
	mock.ExpectExec("INSERT INTO app_meta").WithArgs(abyssShopLoyaltyKey("delver"), "1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE users SET abyss_tokens").WithArgs(int64(18), "delver").WillReturnResult(sqlmock.NewResult(0, 1))
	for range 3 {
		mock.ExpectExec("INSERT INTO user_consumables").WithArgs("delver", sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT abyss_tokens FROM users").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"abyss_tokens"}).AddRow(int64(42)))
	mock.ExpectQuery("SELECT cons_id, remaining_fights").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"cons_id", "remaining_fights"}))
	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/shop/bundle", strings.NewReader(`{"bundle":"delver_supply","quoted_cost":18}`))
	response := httptest.NewRecorder()
	server.handleAbyssShopBundleBuy(response, request, "delver")
	if body := response.Body.String(); !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"loyalty_punches":1`) {
		t.Fatalf("bundle response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssEconomyProgramHelpers(t *testing.T) {
	if got := abyssAuctionSalesTax(100); got != 5 {
		t.Fatalf("auction tax = %d, want 5", got)
	}
	if got := abyssPactTithe([]string{"tithe"}, 999); got != 99 {
		t.Fatalf("pact tithe = %d, want 99", got)
	}
	if got := abyssPactLuck([]string{"tithe"}, 80); got != 88 {
		t.Fatalf("pact luck = %d, want 88", got)
	}
	if len(abyssShopBundles) != 2 || abyssShopGiftFeeGold <= 0 || abyssSeasonExchangeCost <= 0 {
		t.Fatal("shop program constants are incomplete")
	}
}

func TestDuplicateLoreConvertsToTokens(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectExec("INSERT INTO abyss_lore_unlocked").WithArgs("reader", 4).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET abyss_tokens=abyss_tokens+$1 WHERE client_uid=$2")).WithArgs(abyssDuplicateLoreTokens, "reader").WillReturnResult(sqlmock.NewResult(0, 1))
	unlocked, tokens, err := grantAbyssLoreFragment(database, "reader", 4)
	if err != nil || unlocked || tokens != abyssDuplicateLoreTokens {
		t.Fatalf("duplicate lore = unlocked %v tokens %d err %v", unlocked, tokens, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssShopProgramAssetsAreComponentized(t *testing.T) {
	partial, err := webAssets.ReadFile("webassets/abyss_shop_program.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_shop_program.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{`define "abyssShopProgram"`, "shopWalletGold", "abyssGiftCreateForm", "abyssSeasonExchange", "10th token-priced purchase"} {
		if !strings.Contains(string(partial), token) {
			t.Errorf("shop program partial missing %q", token)
		}
	}
	for _, token := range []string{".ab-shop-program-grid", "@media(max-width:900px)", "forced-colors", "prefers-reduced-motion"} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("shop program stylesheet missing %q", token)
		}
	}
}
