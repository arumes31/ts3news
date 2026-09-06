package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"ts3news/internal/leveling"
)

func TestShopExchangeBalanceChargesThousandTimesOldRate(t *testing.T) {
	t.Parallel()
	testShopExchangeBalancePurchase(t, 100_000, 20, 19_999, 90_000, 21)
}

func TestShopExchangeBalanceReturnsGoldBeyondPrestigeBoundary(t *testing.T) {
	t.Parallel()
	capXP := leveling.XPForLevel(PrestigeThreshold)
	// Offer enough gold to buy more than an entire prestige cycle. Only the
	// three XP remaining before prestige should be charged and awarded.
	amount := int64(capXP+100) * 10_000
	testShopExchangeBalancePurchase(t, amount+777, capXP-3, amount, amount+777-30_000, capXP)
}

func TestShopExchangeBalanceOnlyRequiresGoldForClippedPurchase(t *testing.T) {
	t.Parallel()
	capXP := leveling.XPForLevel(PrestigeThreshold)
	testShopExchangeBalancePurchase(t, 30_777, capXP-3, 1_000_000, 777, capXP)
}

func TestShopExchangeBalanceAppliesWeeklyPurchasePrice(t *testing.T) {
	t.Parallel()
	week := shopXPExchangeWeek(time.Now())
	raw := fmt.Sprintf(`{"week":%q,"count":1}`, week)
	testShopExchangeBalancePurchaseWithHistory(t, 100_000, 20, 29_999, 78_000, 22, raw, 2)
}

func TestShopExchangeBalanceResetsPreviousWeekOnSuccessfulPurchase(t *testing.T) {
	t.Parallel()
	testShopExchangeBalancePurchaseWithHistory(t, 100_000, 20, 19_999, 90_000, 21, `{"week":"2020-W01","count":99}`, 1)
}

func TestShopExchangeBalanceRejectsPurchasesAlreadyReadyForPrestige(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		xp   int
	}{
		{name: "at threshold", xp: leveling.XPForLevel(PrestigeThreshold)},
		{name: "legacy overflow", xp: leveling.XPForLevel(PrestigeThreshold) + 1_000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta("SELECT gold, xp FROM users WHERE client_uid=$1 FOR UPDATE")).
				WithArgs("delver").
				WillReturnRows(sqlmock.NewRows([]string{"gold", "xp"}).AddRow(int64(100_000), tc.xp))
			mock.ExpectRollback()

			server := &WebServer{bot: &Bot{DB: database}}
			request := httptest.NewRequest(http.MethodPost, "/api/shop/exchange", strings.NewReader(`{"direction":"gold_to_xp","amount":10000}`))
			response := httptest.NewRecorder()
			server.handleExchangeAPI(response, request, "delver")
			var result struct {
				OK    bool   `json:"ok"`
				Error string `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.OK || !strings.Contains(strings.ToLower(result.Error), "prestige") {
				t.Errorf("expected prestige requirement without changing wallet, response = %s", response.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Error(err)
			}
		})
	}
}

func testShopExchangeBalancePurchase(t *testing.T, initialGold int64, initialXP int, amount, wantGold int64, wantXP int) {
	t.Helper()
	testShopExchangeBalancePurchaseWithHistory(t, initialGold, initialXP, amount, wantGold, wantXP, "", 1)
}

func testShopExchangeBalancePurchaseWithHistory(t *testing.T, initialGold int64, initialXP int, amount, wantGold int64, wantXP int, raw string, wantCount int64) {
	t.Helper()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT gold, xp FROM users WHERE client_uid=$1 FOR UPDATE")).
		WithArgs("delver").
		WillReturnRows(sqlmock.NewRows([]string{"gold", "xp"}).AddRow(initialGold, initialXP))
	query := mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM app_meta WHERE key=$1")).WithArgs("shop_xp_purchases_delver")
	if raw == "" {
		query.WillReturnError(sql.ErrNoRows)
	} else {
		query.WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(raw))
	}
	wantHistory := fmt.Sprintf(`{"week":%q,"count":%d}`, shopXPExchangeWeek(time.Now()), wantCount)
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs("shop_xp_purchases_delver", wantHistory).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET gold=$1, xp=$2, level=$3 WHERE client_uid=$4")).
		WithArgs(wantGold, wantXP, leveling.LevelForXP(wantXP), "delver").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/shop/exchange", strings.NewReader(fmt.Sprintf(`{"direction":"gold_to_xp","amount":%d}`, amount)))
	response := httptest.NewRecorder()
	server.handleExchangeAPI(response, request, "delver")
	var result struct {
		OK    bool  `json:"ok"`
		Gold  int64 `json:"gold"`
		XP    int   `json:"xp"`
		Level int   `json:"level"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Gold != wantGold || result.XP != wantXP || result.Level != leveling.LevelForXP(wantXP) {
		t.Errorf("want gold=%d XP=%d level=%d; response = %s", wantGold, wantXP, leveling.LevelForXP(wantXP), response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestShopExchangeBalanceWeeklyRate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		count int64
		want  int64
	}{
		{name: "first purchase", count: 0, want: 10_000},
		{name: "second purchase", count: 1, want: 11_000},
		{name: "eleventh purchase", count: 10, want: 20_000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := shopXPExchangeRate(tc.count); got != tc.want {
				t.Errorf("rate after %d purchases = %d, want %d", tc.count, got, tc.want)
			}
		})
	}
	if got := shopXPExchangeRate(math.MaxInt64); got < 10_000 {
		t.Errorf("extreme purchase count overflows rate: %d", got)
	}
}

func TestShopExchangeBalanceWeekUsesMondayUTC(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		now  string
		want string
	}{
		{name: "sunday before reset", now: "2026-09-06T23:59:59Z", want: "2026-W36"},
		{name: "monday reset", now: "2026-09-07T00:00:00Z", want: "2026-W37"},
		{name: "local monday is still UTC sunday", now: "2026-09-07T00:30:00+02:00", want: "2026-W36"},
		{name: "ISO year boundary", now: "2027-01-01T00:00:00Z", want: "2026-W53"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, tc.now)
			if err != nil {
				t.Fatal(err)
			}
			if got := shopXPExchangeWeek(now); got != tc.want {
				t.Errorf("week = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestShopExchangeBalanceWeeklyPurchaseReset(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		raw     string
		week    string
		want    int64
		wantErr bool
	}{
		{name: "same week retains count", raw: `{"week":"2026-W36","count":3}`, week: "2026-W36", want: 3},
		{name: "new week resets count", raw: `{"week":"2026-W36","count":3}`, week: "2026-W37", want: 0},
		{name: "invalid JSON fails closed", raw: `{`, week: "2026-W36", wantErr: true},
		{name: "negative count fails closed", raw: `{"week":"2026-W36","count":-1}`, week: "2026-W36", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := shopXPExchangePurchases(tc.raw, tc.week)
			if (err != nil) != tc.wantErr || (!tc.wantErr && got != tc.want) {
				t.Errorf("purchases = %d, error = %v; want %d, error = %v", got, err, tc.want, tc.wantErr)
			}
		})
	}
}

func TestShopExchangeBalanceStaleQuoteDoesNotSpendGold(t *testing.T) {
	t.Parallel()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT gold, xp FROM users WHERE client_uid=$1 FOR UPDATE")).
		WithArgs("delver").
		WillReturnRows(sqlmock.NewRows([]string{"gold", "xp"}).AddRow(int64(100_000), 20))
	week := shopXPExchangeWeek(time.Now())
	history := fmt.Sprintf(`{"week":%q,"count":1}`, week)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM app_meta WHERE key=$1")).
		WithArgs("shop_xp_purchases_delver").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(history))
	mock.ExpectRollback()

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/shop/exchange", strings.NewReader(`{"direction":"gold_to_xp","amount":22000,"rate":10000}`))
	response := httptest.NewRecorder()
	server.handleExchangeAPI(response, request, "delver")
	var result struct {
		OK   bool  `json:"ok"`
		Rate int64 `json:"gold_per_xp"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.OK || result.Rate != 11_000 {
		t.Errorf("stale quote should fail with current price 11000; response = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestShopExchangeBalanceSaveFailuresRollBackPurchaseAndPrice(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		historyFail bool
	}{
		{name: "history write fails", historyFail: true},
		{name: "wallet write fails"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta("SELECT gold, xp FROM users WHERE client_uid=$1 FOR UPDATE")).
				WithArgs("delver").
				WillReturnRows(sqlmock.NewRows([]string{"gold", "xp"}).AddRow(int64(100_000), 20))
			mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM app_meta WHERE key=$1")).
				WithArgs("shop_xp_purchases_delver").
				WillReturnError(sql.ErrNoRows)
			historyWrite := mock.ExpectExec("INSERT INTO app_meta").
				WithArgs("shop_xp_purchases_delver", sqlmock.AnyArg())
			if tc.historyFail {
				historyWrite.WillReturnError(errors.New("injected history failure"))
			} else {
				historyWrite.WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET gold=$1, xp=$2, level=$3 WHERE client_uid=$4")).
					WithArgs(int64(90_000), 21, leveling.LevelForXP(21), "delver").
					WillReturnError(errors.New("injected wallet failure"))
			}
			mock.ExpectRollback()

			server := &WebServer{bot: &Bot{DB: database}}
			request := httptest.NewRequest(http.MethodPost, "/api/shop/exchange", strings.NewReader(`{"direction":"gold_to_xp","amount":10000}`))
			response := httptest.NewRecorder()
			server.handleExchangeAPI(response, request, "delver")
			var result struct {
				OK bool `json:"ok"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.OK {
				t.Errorf("failed save reported success: %s", response.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Error(err)
			}
		})
	}
}
