package bot

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssPouchCapsAreBounded(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		level, stack, carry int
	}{{-1, 5, 8}, {0, 5, 8}, {1, 6, 9}, {2, 7, 10}, {3, 8, 11}, {99, 8, 11}} {
		stack, carry := abyssPouchCaps(test.level)
		if stack != test.stack || carry != test.carry {
			t.Errorf("level %d caps = %d/%d, want %d/%d", test.level, stack, carry, test.stack, test.carry)
		}
	}
}

func TestUpgradeAbyssPouchIsAtomic(t *testing.T) {
	tests := []struct {
		name      string
		owned     bool
		level     int
		gold      any
		wantLevel int
		wantGold  int64
		wantErr   error
	}{
		{name: "rank one", owned: true, level: 0, gold: int64(750_000), wantLevel: 1, wantGold: 750_000},
		{name: "maxed", owned: true, level: 3, wantErr: errAbyssPouchMaxed},
		{name: "insufficient funds", owned: true, level: 1, gold: sql.ErrNoRows, wantErr: errAbyssPouchFunds},
		{name: "missing pouch", wantErr: errAbyssPouchMissing},
	}
	for _, test := range tests {
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
			gearQuery := mock.ExpectQuery("SELECT 1 FROM user_gear").WithArgs("delver")
			if test.owned {
				gearQuery.WillReturnRows(sqlmock.NewRows([]string{"found"}).AddRow(1))
				mock.ExpectExec("INSERT INTO abyss_consumable_pouches").WithArgs("delver").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectQuery("SELECT level FROM abyss_consumable_pouches").WithArgs("delver").
					WillReturnRows(sqlmock.NewRows([]string{"level"}).AddRow(test.level))
				if test.level < abyssPouchMaxLevel {
					goldQuery := mock.ExpectQuery("UPDATE users SET gold=gold-\\$1").
						WithArgs(abyssPouchUpgradeCosts[test.level], "delver")
					if goldErr, ok := test.gold.(error); ok {
						goldQuery.WillReturnError(goldErr)
					} else {
						goldQuery.WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(test.gold))
						mock.ExpectExec("UPDATE abyss_consumable_pouches").WithArgs(test.level+1, "delver").
							WillReturnResult(sqlmock.NewResult(0, 1))
					}
				}
			} else {
				gearQuery.WillReturnError(sql.ErrNoRows)
				mock.ExpectQuery("SELECT 1 FROM user_inventory").WithArgs("delver").WillReturnError(sql.ErrNoRows)
			}

			level, gold, gotErr := upgradeAbyssPouch(tx, "delver")
			if !errors.Is(gotErr, test.wantErr) {
				t.Fatalf("error = %v, want %v", gotErr, test.wantErr)
			}
			if level != test.wantLevel || gold != test.wantGold {
				t.Fatalf("result = level %d, gold %d; want %d, %d", level, gold, test.wantLevel, test.wantGold)
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

func TestAbyssPouchUpgradeRoutesMigrationAndUIContract(t *testing.T) {
	t.Parallel()

	routes, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	entry, err := os.ReadFile("web_abyss.go")
	if err != nil {
		t.Fatal(err)
	}
	loot, err := os.ReadFile("web_abyss_loot2.go")
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../db/migrations/0097_abyss_pouch_upgrades.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	page, err := webAssets.ReadFile("webassets/inventory.html")
	if err != nil {
		t.Fatal(err)
	}
	combined := string(routes) + string(entry) + string(loot) + string(migration) + string(page)
	for _, required := range []string{
		`/api/inventory/pouch/upgrade`, `abyssPouchCaps(s.bot.abyssPouchLevel(uid))`,
		`stackLimit := b.abyssConsumableStackLimit(uid)`, `CHECK (level BETWEEN 0 AND 3)`,
		`GREATEST(user_consumables.remaining_fights`,
		`+1 Abyss loot stack · +1 equipped run carry`, `PERMANENT STORAGE TAILORING`,
	} {
		if !strings.Contains(combined, required) {
			t.Errorf("pouch upgrade contract is missing %q", required)
		}
	}
}
