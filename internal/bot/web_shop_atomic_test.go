package bot

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"ts3news/internal/leveling"
)

func TestShopExchangeLocksWalletBeforeAbsoluteBalanceWrite(t *testing.T) {
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
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs("shop_xp_purchases_delver", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET gold=$1, xp=$2, level=$3 WHERE client_uid=$4")).
		WithArgs(int64(90_000), 21, leveling.LevelForXP(21), "delver").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/shop/exchange", strings.NewReader(`{"direction":"gold_to_xp","amount":19999}`))
	response := httptest.NewRecorder()
	server.handleExchangeAPI(response, request, "delver")

	if body := response.Body.String(); !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"gold":90000`) || !strings.Contains(body, `"xp":21`) {
		t.Fatalf("exchange response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestShopExchangeValidationRollsBackLockedWallet(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT gold, xp FROM users WHERE client_uid=$1 FOR UPDATE")).
		WithArgs("delver").
		WillReturnRows(sqlmock.NewRows([]string{"gold", "xp"}).AddRow(int64(5), 20))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM app_meta WHERE key=$1")).
		WithArgs("shop_xp_purchases_delver").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/shop/exchange", strings.NewReader(`{"direction":"gold_to_xp","amount":10000}`))
	response := httptest.NewRecorder()
	server.handleExchangeAPI(response, request, "delver")

	if body := response.Body.String(); !strings.Contains(body, `"ok":false`) || !strings.Contains(body, "not enough gold") {
		t.Fatalf("exchange response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestShopBuyResponseReportsAutoEquip(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	seed, _ := shopWindow(time.Now())
	item := stockForSeed(seed, nil)[0]
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gold FROM users.*FOR UPDATE").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(int64(25_000_000)))
	mock.ExpectQuery("SELECT value FROM app_meta").WithArgs(shopBuffKey("delver")).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("UPDATE users SET gold = gold -").
		WithArgs(item.Price, "delver").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_gear").
		WithArgs("delver", item.Slot).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT gear_id, durability, item_data FROM user_gear").
		WithArgs("delver", item.Slot).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO user_gear").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT gold FROM users").
		WithArgs("delver").
		WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(int64(9_000_000)))
	mock.ExpectCommit()

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/shop/buy", strings.NewReader(`{"id":"`+item.ID+`"}`))
	response := httptest.NewRecorder()
	server.handleBuyAPI(response, request, "delver")

	if body := response.Body.String(); !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"equipped":true`) {
		t.Fatalf("purchase response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
