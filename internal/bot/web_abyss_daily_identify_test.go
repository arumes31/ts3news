package bot

import (
	"context"
	"database/sql"
	"errors"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

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

func TestAbyssDailyIdentifyQuoteWaivesFixedCost(t *testing.T) {
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

func TestAbyssPaidIdentifyQuoteDoesNotRevealRarity(t *testing.T) {
	for _, rarity := range []content.Rarity{content.RarityCommon, content.RarityCelestial} {
		t.Run(rarity.String(), func(t *testing.T) {
			server, mock, done := newForge2TestServer(t)
			defer done()
			uid := "paid-identify-" + rarity.String()
			mock.ExpectQuery("SELECT NOT EXISTS.*app_meta").WithArgs(abyssDailyIdentifyKey(uid)).
				WillReturnRows(sqlmock.NewRows([]string{"available"}).AddRow(false))

			cost, minimum, maximum, err := server.resolveAbyssForgeQuoteCost(
				context.Background(), uid, "identify", &content.Gear{Rarity: rarity, Unidentified: true}, nil,
			)
			if err != nil {
				t.Fatalf("resolve identify quote: %v", err)
			}
			if cost.Gold != abyssIdentifyCost || minimum.Gold != abyssIdentifyCost || maximum.Gold != abyssIdentifyCost {
				t.Fatalf("paid identify quote costs = %d/%d/%d, want fixed %d", cost.Gold, minimum.Gold, maximum.Gold, abyssIdentifyCost)
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
	expectDailyIdentifyClaim(mock, uid, true)
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
	expectDailyIdentifyClaim(mock, uid, false)
	mock.ExpectExec("UPDATE user_inventory SET item_data=").WithArgs(sqlmock.AnyArg(), int64(99), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE users SET gold = gold -").WithArgs(int64(abyssIdentifyCost), uid).
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
	expectDailyIdentifyClaim(mock, uid, true)
	mock.ExpectExec("UPDATE users SET gold = gold -").WithArgs(int64(abyssIdentifyCost), uid).
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
		!strings.Contains(body, `"cost":1000`) {
		t.Fatalf("identify-all response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestHandleAbyssIdentifyRollsBackClaimWhenItemWriteFails(t *testing.T) {
	server, mock, done := newForge2TestServer(t)
	defer done()
	uid := "daily-identify-rollback"

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_inventory").WithArgs(int64(103), uid).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow("U_LEG_2", `{"unidentified":true}`))
	expectDailyIdentifyClaim(mock, uid, true)
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
	expectDailyIdentifyClaim(mock, uid, false)
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
