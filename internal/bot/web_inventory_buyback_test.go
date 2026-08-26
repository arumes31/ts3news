package bot

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssVendorBuybackCost(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		sale int64
		want int64
	}{{0, 0}, {1, 2}, {10, 11}, {11, 13}, {100, 110}, {math.MaxInt64, math.MaxInt64}} {
		if got := abyssVendorBuybackCost(test.sale); got != test.want {
			t.Errorf("buyback cost for %d = %d, want %d", test.sale, got, test.want)
		}
	}
}

func TestRecordVendorBuybackStoresExactItemAndBoundsLedger(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	tx, err := database.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	acquiredAt := time.Date(2026, 8, 20, 12, 30, 0, 0, time.UTC)
	itemData := sql.NullString{String: `{"rarity":4,"name":"Exact Relic"}`, Valid: true}
	mock.ExpectExec("INSERT INTO abyss_vendor_buybacks").
		WithArgs("delver", "ABYSS_RELIC", 37, itemData, acquiredAt, int64(1_001), int64(1_102)).
		WillReturnResult(sqlmock.NewResult(44, 1))
	mock.ExpectExec("DELETE FROM abyss_vendor_buybacks").
		WithArgs("delver", abyssVendorBuybackLimit).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := recordVendorBuyback(tx, "delver", "ABYSS_RELIC", 37, itemData, acquiredAt, 1_001); err != nil {
		t.Fatal(err)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBuyBackVendorItemIsAtomic(t *testing.T) {
	for _, test := range []struct {
		name      string
		gold      any
		wantGold  int64
		wantErr   error
		wantWrite bool
	}{
		{name: "restores exact item", gold: int64(7_890), wantGold: 7_890, wantWrite: true},
		{name: "insufficient funds", gold: sql.ErrNoRows, wantErr: errVendorBuybackFunds},
	} {
		t.Run(test.name, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()

			mock.ExpectBegin()
			tx, err := database.BeginTx(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			acquiredAt := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
			itemData := sql.NullString{String: `{"name":"Returned Blade"}`, Valid: true}
			mock.ExpectQuery("SELECT gear_id,durability,item_data,acquired_at,buyback_cost").
				WithArgs(int64(9), "delver").WillReturnRows(sqlmock.NewRows([]string{
				"gear_id", "durability", "item_data", "acquired_at", "buyback_cost",
			}).AddRow("ABYSS_BLADE", 42, itemData, acquiredAt, int64(1_100)))
			goldQuery := mock.ExpectQuery("UPDATE users SET gold=gold-\\$1").WithArgs(int64(1_100), "delver")
			if goldErr, ok := test.gold.(error); ok {
				goldQuery.WillReturnError(goldErr)
			} else {
				goldQuery.WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(test.gold))
			}
			if test.wantWrite {
				mock.ExpectExec("INSERT INTO user_inventory").
					WithArgs("delver", "ABYSS_BLADE", 42, itemData, acquiredAt).
					WillReturnResult(sqlmock.NewResult(55, 1))
				mock.ExpectExec("DELETE FROM abyss_vendor_buybacks").
					WithArgs(int64(9), "delver").WillReturnResult(sqlmock.NewResult(0, 1))
			}

			gotGold, gotErr := buyBackVendorItem(tx, "delver", 9)
			if !errors.Is(gotErr, test.wantErr) {
				t.Fatalf("error = %v, want %v", gotErr, test.wantErr)
			}
			if gotGold != test.wantGold {
				t.Fatalf("gold = %d, want %d", gotGold, test.wantGold)
			}
			mock.ExpectRollback()
			if err := tx.Rollback(); err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAbyssVendorBuybackRoutesMigrationAndUIContract(t *testing.T) {
	t.Parallel()

	routes, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	saleSource, err := os.ReadFile("web_pages.go")
	if err != nil {
		t.Fatal(err)
	}
	buybackSource, err := os.ReadFile("web_inventory_buyback.go")
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../db/migrations/0096_abyss_vendor_buyback.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	page, err := webAssets.ReadFile("webassets/inventory.html")
	if err != nil {
		t.Fatal(err)
	}
	combined := string(routes) + string(saleSource) + string(buybackSource) + string(migration) + string(page)
	for _, required := range []string{
		`/api/inventory/buyback`, `recordVendorBuyback(tx`, `abyss_vendor_buybacks`,
		`ORDER BY sold_at DESC,id DESC OFFSET $2`, `sale price + 10% handling`,
		`Exact item restored`, `Includes the disclosed 10% vendor handling fee`,
	} {
		if !strings.Contains(combined, required) {
			t.Errorf("vendor buyback contract is missing %q", required)
		}
	}
}
