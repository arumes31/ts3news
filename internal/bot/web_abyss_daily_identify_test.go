package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func expectDailyIdentifyCharge(mock sqlmock.Sqlmock, uid string, claimed bool) {
	expectDailyIdentifyBalanceAndClaim(mock, uid, 500, claimed)
}

func expectDailyIdentifyBalanceAndClaim(mock sqlmock.Sqlmock, uid string, gold int64, claimed bool) {
	mock.ExpectQuery("SELECT gold FROM users.*FOR UPDATE").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(gold))
	expectDailyIdentifyClaim(mock, uid, claimed)
}

func expectDailyIdentifyClaim(mock sqlmock.Sqlmock, uid string, claimed bool) {
	rows := sqlmock.NewRows([]string{"claimed"})
	if claimed {
		rows.AddRow(true)
	}
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO app_meta (key, value) VALUES ($1, (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')::date::text)")).
		WithArgs(abyssDailyIdentifyKey(uid)).WillReturnRows(rows)
}

func TestAbyssDailyIdentifyAvailableUsesUTCDate(t *testing.T) {
	server, mock, done := newForge2TestServer(t)
	defer done()
	uid := "daily-identify-state"

	mock.ExpectQuery(regexp.QuoteMeta("SELECT NOT EXISTS(SELECT 1 FROM app_meta WHERE key=$1 AND value=(CURRENT_TIMESTAMP AT TIME ZONE 'UTC')::date::text)")).
		WithArgs(abyssDailyIdentifyKey(uid)).
		WillReturnRows(sqlmock.NewRows([]string{"available"}).AddRow(true))

	available, err := abyssDailyIdentifyAvailable(context.Background(), server.bot.DB, uid)
	if err != nil || !available {
		t.Fatalf("available = %v, err = %v", available, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestAbyssDailyIdentifyQuoteWaivesTierCost(t *testing.T) {
	server, mock, done := newForge2TestServer(t)
	defer done()
	uid := "daily-identify-quote"
	mock.ExpectQuery("SELECT NOT EXISTS.*app_meta").WithArgs(abyssDailyIdentifyKey(uid)).
		WillReturnRows(sqlmock.NewRows([]string{"available"}).AddRow(true))

	gear := &content.Gear{Rarity: content.RarityLegendary}
	cost, minimum, maximum, err := server.resolveAbyssForgeQuoteCost(
		context.Background(),
		uid,
		"identify",
		gear,
		nil,
	)
	if err != nil {
		t.Fatalf("resolve identify quote: %v", err)
	}
	if cost.Gold != 0 || minimum.Gold != 0 || maximum.Gold != 0 {
		t.Fatalf("daily identify quote costs = %d/%d/%d, want all zero", cost.Gold, minimum.Gold, maximum.Gold)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestAbyssPaidIdentifyQuoteUsesTierCost(t *testing.T) {
	for _, rarity := range []content.Rarity{content.RarityCommon, content.RarityCelestial} {
		t.Run(rarity.String(), func(t *testing.T) {
			server, mock, done := newForge2TestServer(t)
			defer done()
			uid := "paid-identify-" + rarity.String()
			mock.ExpectQuery("SELECT NOT EXISTS.*app_meta").WithArgs(abyssDailyIdentifyKey(uid)).
				WillReturnRows(sqlmock.NewRows([]string{"available"}).AddRow(false))
			mock.ExpectQuery("SELECT gold FROM users").WithArgs(uid).
				WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(500))

			cost, minimum, maximum, err := server.resolveAbyssForgeQuoteCost(
				context.Background(), uid, "identify", &content.Gear{Rarity: rarity, Unidentified: true}, nil,
			)
			if err != nil {
				t.Fatalf("resolve identify quote: %v", err)
			}
			want := identifyGearCost(rarity)
			if cost.Gold != want || minimum.Gold != want || maximum.Gold != want {
				t.Fatalf("paid identify quote costs = %d/%d/%d, want %d", cost.Gold, minimum.Gold, maximum.Gold, want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("database expectations: %v", err)
			}
		})
	}
}

func TestHandleAbyssIdentifyClaimsFreeUseWithItemCommit(t *testing.T) {
	server, mock, done := newForge2TestServer(t)
	defer done()
	uid := "daily-identify-free"

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").WithArgs(int64(98), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", `{"unidentified":true}`))
	expectDailyIdentifyCharge(mock, uid, true)
	mock.ExpectExec("UPDATE user_inventory SET item_data=").WithArgs(sqlmock.AnyArg(), int64(98), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT gold FROM users").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(12345))

	recorder := postForge2(t, server.handleAbyssIdentify, `{"inv_id":98}`, uid)
	if body := recorder.Body.String(); !strings.Contains(body, `"ok":true`) ||
		!strings.Contains(body, `"daily_free":true`) || !strings.Contains(body, `"cost":0`) {
		t.Fatalf("identify response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestHandleAbyssIdentifyChargesAfterFreeUse(t *testing.T) {
	server, mock, done := newForge2TestServer(t)
	defer done()
	uid := "daily-identify-paid"

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").WithArgs(int64(99), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", `{"unidentified":true}`))
	expectDailyIdentifyCharge(mock, uid, false)
	mock.ExpectExec("UPDATE user_inventory SET item_data=").WithArgs(sqlmock.AnyArg(), int64(99), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE users SET gold = gold -").WithArgs(identifyGearCost(content.RarityLegendary), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT gold FROM users").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(2345))

	recorder := postForge2(t, server.handleAbyssIdentify, `{"inv_id":99}`, uid)
	if body := recorder.Body.String(); !strings.Contains(body, `"ok":true`) ||
		!strings.Contains(body, `"daily_free":false`) || strings.Contains(body, `"cost":0`) {
		t.Fatalf("identify response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestHandleAbyssIdentifyAllDiscountsExactlyOneItem(t *testing.T) {
	server, mock, done := newForge2TestServer(t)
	defer done()
	uid := "daily-identify-all"

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, gear_id, item_data FROM user_inventory").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"id", "gear_id", "item_data"}).
			AddRow(101, "U_LEG_2", `{"unidentified":true}`).
			AddRow(102, "U_LEG_2", `{"unidentified":true}`))
	mock.ExpectQuery("SELECT slot, gear_id, item_data FROM user_gear").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"slot", "gear_id", "item_data"}))
	expectDailyIdentifyCharge(mock, uid, true)
	mock.ExpectExec("UPDATE users SET gold = gold -").WithArgs(identifyGearCost(content.RarityLegendary), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE user_inventory SET item_data=").WithArgs(sqlmock.AnyArg(), int64(101), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE user_inventory SET item_data=").WithArgs(sqlmock.AnyArg(), int64(102), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT gold FROM users").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(9000))

	recorder := postForge2(t, server.handleAbyssIdentifyAll, `{}`, uid)
	if body := recorder.Body.String(); !strings.Contains(body, `"daily_free":true`) ||
		!strings.Contains(body, `"cost":35`) {
		t.Fatalf("identify-all response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestHandleAbyssIdentifyCapsChargeAtBalance(t *testing.T) {
	for _, gold := range []int64{0, 7} {
		t.Run(fmt.Sprintf("gold_%d", gold), func(t *testing.T) {
			server, mock, done := newForge2TestServer(t)
			defer done()
			uid := "identify-low-gold"
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").WithArgs(int64(99), uid).
				WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", `{"unidentified":true}`))
			expectDailyIdentifyBalanceAndClaim(mock, uid, gold, false)
			mock.ExpectExec("UPDATE user_inventory SET item_data=").WithArgs(sqlmock.AnyArg(), int64(99), uid).
				WillReturnResult(sqlmock.NewResult(0, 1))
			if gold > 0 {
				mock.ExpectExec("UPDATE users SET gold = gold -").WithArgs(gold, uid).
					WillReturnResult(sqlmock.NewResult(0, 1))
			}
			mock.ExpectCommit()
			mock.ExpectQuery("SELECT gold FROM users").WithArgs(uid).
				WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(0))
			recorder := postForge2(t, server.handleAbyssIdentify, `{"inv_id":99}`, uid)
			if body := recorder.Body.String(); !strings.Contains(body, `"ok":true`) || !strings.Contains(body, fmt.Sprintf(`"cost":%d`, gold)) {
				t.Fatalf("identify response = %s", body)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAbyssIdentifyAllQuoteDiscountsFirstItemTierCost(t *testing.T) {
	server, mock, done := newForge2TestServer(t)
	defer done()
	uid := "identify-batch-quote"
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory.*ORDER BY id").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).
			AddRow("U_LEG_2", `{"unidentified":true,"rarity":0}`).
			AddRow("U_LEG_2", `{"unidentified":true,"rarity":4}`))
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_gear.*ORDER BY slot").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).
			AddRow("U_LEG_2", `{"unidentified":true,"rarity":8}`))
	mock.ExpectQuery("SELECT NOT EXISTS.*app_meta").WithArgs(abyssDailyIdentifyKey(uid)).
		WillReturnRows(sqlmock.NewRows([]string{"available"}).AddRow(true))
	mock.ExpectQuery("SELECT gold FROM users").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(500))
	cost, minimum, maximum, err := server.resolveAbyssForgeQuoteCost(context.Background(), uid, "identify_all", nil, nil)
	if err != nil || cost.Gold != 135 || minimum.Gold != 135 || maximum.Gold != 135 {
		t.Fatalf("quote = %+v/%+v/%+v, err = %v", cost, minimum, maximum, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleAbyssIdentifyRollsBackClaimWhenItemWriteFails(t *testing.T) {
	server, mock, done := newForge2TestServer(t)
	defer done()
	uid := "daily-identify-rollback"

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").WithArgs(int64(103), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", `{"unidentified":true}`))
	expectDailyIdentifyCharge(mock, uid, true)
	mock.ExpectExec("UPDATE user_inventory SET item_data=").WithArgs(sqlmock.AnyArg(), int64(103), uid).
		WillReturnError(errors.New("write failed"))
	mock.ExpectRollback()

	recorder := postForge2(t, server.handleAbyssIdentify, `{"inv_id":103}`, uid)
	if !strings.Contains(recorder.Body.String(), `"error":"db"`) {
		t.Fatalf("identify response = %s", recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestHandleAbyssIdentifyDoesNotClaimInvalidItem(t *testing.T) {
	server, mock, done := newForge2TestServer(t)
	defer done()
	uid := "daily-identify-invalid"

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").WithArgs(int64(98), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", `{}`))
	mock.ExpectRollback()

	recorder := postForge2(t, server.handleAbyssIdentify, `{"inv_id":98}`, uid)
	if !strings.Contains(recorder.Body.String(), "already identified") {
		t.Fatalf("identify response = %s", recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestDailyIdentifyRejectsQuoteAfterConcurrentClaim(t *testing.T) {
	server, mock, done := newForge2TestServer(t)
	defer done()
	uid := "daily-identify-stale"
	quotedGold := int64(0)
	token, err := server.signForgeClaims(abyssForgeQuoteClaims{
		UID: uid, Operation: "identify", QuotedGold: &quotedGold, ExpiresUnix: time.Now().Add(time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("sign quote: %v", err)
	}

	mock.ExpectBegin()
	expectDailyIdentifyCharge(mock, uid, false)
	mock.ExpectRollback()
	req := httptest.NewRequest("POST", "/api/abyss/identify", strings.NewReader(`{}`))
	req.Header.Set(abyssForgeQuoteHeader, token)
	recorder := httptest.NewRecorder()
	tx := mustBeginTestTx(t, server.bot.DB)
	_, _, ok := server.dailyIdentifyCharge(recorder, req, tx, uid, 10_000, 10_000)
	if ok || !strings.Contains(recorder.Body.String(), errAbyssDailyIdentifyQuoteStale.Error()) {
		t.Fatalf("stale quote response = %s", recorder.Body.String())
	}
	_ = tx.Rollback()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func mustBeginTestTx(t *testing.T, db *sql.DB) *sql.Tx {
	t.Helper()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	return tx
}

func TestClaimAbyssDailyIdentifyPropagatesDatabaseFailure(t *testing.T) {
	server, mock, done := newForge2TestServer(t)
	defer done()
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO app_meta").WillReturnError(errors.New("database unavailable"))
	mock.ExpectRollback()
	tx := mustBeginTestTx(t, server.bot.DB)
	claimed, err := claimAbyssDailyIdentify(context.Background(), tx, "failure")
	if err == nil || claimed {
		t.Fatalf("claimed = %v, err = %v", claimed, err)
	}
	_ = tx.Rollback()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}
