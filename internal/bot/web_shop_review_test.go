package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"ts3news/internal/content"
	"ts3news/internal/leveling"
)

func TestShopQuoteExplainsLevelLossAndRounding(t *testing.T) {
	xp := leveling.XPForLevel(100)
	q, err := makeShopExchangeQuote("xp_to_gold", int64(xp), 1000, xp, goldPerXP)
	if err != nil {
		t.Fatal(err)
	}
	if !q.LevelLoss || q.Before.Level != 100 || q.After.Level != 1 || q.After.XP != xp%2 || q.Spend+q.Unspent != int64(xp) || q.After.Gold != 1000+q.Gain {
		t.Fatalf("incorrect progression preview: %+v", q)
	}
}

func TestShopQuoteMatchesPrestigeCapAndNextPrice(t *testing.T) {
	xp := leveling.XPForLevel(PrestigeThreshold) - 3
	q, err := makeShopExchangeQuote("gold_to_xp", 1_000_000, 33_777, xp, 11_000)
	if err != nil {
		t.Fatal(err)
	}
	if q.Spend != 33_000 || q.Gain != 3 || q.After.Gold != 777 || q.After.XP != xp+3 || q.NextRate != 12_000 || q.Unspent != 967_000 {
		t.Fatalf("incorrect capped preview: %+v", q)
	}
}

func TestShopQuoteRejectsInvalidOrUnaffordableInputs(t *testing.T) {
	for _, tc := range []struct {
		direction    string
		amount, gold int64
		xp           int
	}{
		{"gold_to_xp", 0, 100_000, 10}, {"gold_to_xp", 10_000, 999, 10},
		{"xp_to_gold", 20, 0, 10}, {"xp_to_gold", 2, math.MaxInt64, 10},
		{"unknown", 100, 100_000, 100},
	} {
		if _, err := makeShopExchangeQuote(tc.direction, tc.amount, tc.gold, tc.xp, goldPerXP); err == nil {
			t.Errorf("accepted invalid quote: %+v", tc)
		}
	}
}

func TestShopExchangePreviewDoesNotWriteWalletOrHistory(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gold, xp FROM users").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"gold", "xp"}).AddRow(100_000, 20))
	mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("shop_xp_purchases_delver").WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()
	server := &WebServer{bot: &Bot{DB: database}}
	response := httptest.NewRecorder()
	server.handleExchangeAPI(response, httptest.NewRequest(http.MethodPost, "/api/shop/exchange", strings.NewReader(`{"direction":"gold_to_xp","amount":10000,"preview":true}`)), "delver")
	var result struct {
		OK    bool              `json:"ok"`
		Quote shopExchangeQuote `json:"quote"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Quote.After.Gold != 90_000 || result.Quote.NextRate != 11_000 {
		t.Fatalf("preview = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestShopExchangeRejectsChangedWalletBeforeSpending(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gold, xp FROM users").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"gold", "xp"}).AddRow(2000, 500))
	mock.ExpectRollback()
	server := &WebServer{bot: &Bot{DB: database}}
	response := httptest.NewRecorder()
	server.handleExchangeAPI(response, httptest.NewRequest(http.MethodPost, "/api/shop/exchange", strings.NewReader(`{"direction":"xp_to_gold","amount":100,"expected":{"gold":1000,"xp":500},"confirm_level_loss":true}`)), "delver")
	if !strings.Contains(response.Body.String(), `"review_required":true`) {
		t.Fatalf("stale preview accepted: %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestShopComparisonIncludesLostStatsAndSpecials(t *testing.T) {
	current := content.Gear{Name: "Forged helm", Slot: content.SlotHead, Stats: content.Stats{STR: 80, DEF: 20}, Special: content.EffectVampiric, XPMultiplier: 1.2}
	candidate := content.Gear{Name: "New helm", Slot: content.SlotHead, Stats: content.Stats{STR: 100}, Special: content.EffectQuick, XPMultiplier: 1.1}
	comparison := shopGearComparison(candidate, map[string]content.Gear{string(content.SlotHead): current})
	changes := map[string]int{}
	for _, stat := range comparison.Stats {
		changes[stat.Label] = stat.Value
	}
	if comparison.EquippedName != current.Name || changes["STR"] != 20 || changes["DEF"] != -20 || comparison.XPBonusDelta != -10 || len(comparison.AddedSpecials) != 1 || len(comparison.RemovedSpecials) != 1 {
		t.Fatalf("comparison lost tradeoffs: %+v", comparison)
	}
}

func TestShopExchangeRequiresReviewOfLevelLoss(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gold, xp FROM users").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"gold", "xp"}).AddRow(1000, 500))
	mock.ExpectRollback()
	server := &WebServer{bot: &Bot{DB: database}}
	response := httptest.NewRecorder()
	server.handleExchangeAPI(response, httptest.NewRequest(http.MethodPost, "/api/shop/exchange", strings.NewReader(`{"direction":"xp_to_gold","amount":500,"expected":{"gold":1000,"xp":500}}`)), "delver")
	if !strings.Contains(response.Body.String(), `"review_required":true`) {
		t.Fatalf("level loss accepted without review: %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestShopExchangeCommitFailureIsUnconfirmed(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gold, xp FROM users").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"gold", "xp"}).AddRow(1000, 500))
	mock.ExpectExec("UPDATE users SET gold=").WithArgs(int64(1250), 0, 1, "delver").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errors.New("connection lost at commit"))
	server := &WebServer{bot: &Bot{DB: database}}
	response := httptest.NewRecorder()
	server.handleExchangeAPI(response, httptest.NewRequest(http.MethodPost, "/api/shop/exchange", strings.NewReader(`{"direction":"xp_to_gold","amount":500,"expected":{"gold":1000,"xp":500},"confirm_level_loss":true}`)), "delver")
	if !strings.Contains(response.Body.String(), `"unconfirmed":true`) {
		t.Fatalf("ambiguous commit treated as retryable: %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestShopBuyRejectsChangedReviewedBalanceBeforeDeliveringItem(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	seed, _ := shopWindow(time.Now())
	item := stockForSeed(seed, nil)[0]
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gold FROM users.*FOR UPDATE").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(int64(24_000_000)))
	mock.ExpectRollback()
	server := &WebServer{bot: &Bot{DB: database}}
	response := httptest.NewRecorder()
	server.handleBuyAPI(response, httptest.NewRequest(http.MethodPost, "/api/shop/buy", strings.NewReader(`{"id":"`+item.ID+`","expected_gold":25000000}`)), "delver")
	if !strings.Contains(response.Body.String(), `"review_required":true`) {
		t.Fatalf("stale purchase accepted: %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
