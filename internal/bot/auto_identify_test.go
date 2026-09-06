package bot

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"ts3news/internal/content"
)

func TestIdentifyGearCost(t *testing.T) {
	for rarity, want := range []int64{1, 5, 10, 20, 35, 50, 65, 80, 100} {
		if got := identifyGearCost(content.Rarity(rarity)); got != want {
			t.Errorf("tier %d: cost = %d, want %d", rarity, got, want)
		}
	}
	if identifyGearCost(-1) != 1 || identifyGearCost(999) != 100 {
		t.Fatal("unknown rarity must stay within 1–100 gold")
	}
}

type identifiedGearJSON struct{}

func TestAutoIdentifySkipsAlreadyIdentifiedItems(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for range 2 {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, gear_id, item_data FROM user_inventory.*unidentified.*FOR UPDATE").WithArgs("player").
			WillReturnRows(sqlmock.NewRows([]string{"id", "gear_id", "item_data"}))
		mock.ExpectQuery("SELECT slot, gear_id, item_data FROM user_gear.*unidentified.*FOR UPDATE").WithArgs("player").
			WillReturnRows(sqlmock.NewRows([]string{"slot", "gear_id", "item_data"}))
		mock.ExpectRollback()
		count, cost, err := (&Bot{DB: db}).autoIdentifyItems(context.Background(), "player")
		if err != nil || count != 0 || cost != 0 {
			t.Fatalf("count=%d cost=%d err=%v", count, cost, err)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAutoIdentifyRollsBackFailedExistingItemWrite(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, gear_id, item_data FROM user_inventory").WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"id", "gear_id", "item_data"}).AddRow(1, "secret", `{"ID":"secret","Rarity":8,"unidentified":true}`))
	mock.ExpectQuery("SELECT slot, gear_id, item_data FROM user_gear").WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"slot", "gear_id", "item_data"}))
	mock.ExpectQuery("SELECT gold FROM users.*FOR UPDATE").WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(200))
	expectDailyIdentifyClaim(mock, "player", false)
	mock.ExpectExec("UPDATE users SET gold = gold -").WithArgs(int64(100), "player").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE user_inventory SET item_data=jsonb_set").WithArgs(int64(1), "player").
		WillReturnError(errors.New("write failed"))
	mock.ExpectRollback()
	count, cost, err := (&Bot{DB: db}).autoIdentifyItems(context.Background(), "player")
	if err == nil || count != 0 || cost != 0 {
		t.Fatalf("count=%d cost=%d err=%v", count, cost, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func (identifiedGearJSON) Match(value driver.Value) bool {
	raw, ok := value.(string)
	var gear content.Gear
	return ok && json.Unmarshal([]byte(raw), &gear) == nil && !gear.Unidentified && gear.ID == "secret"
}

func TestStoreAutoIdentifiedDrop(t *testing.T) {
	for _, test := range []struct {
		name         string
		gold, charge int64
		free, fail   bool
	}{
		{name: "paid", gold: 200, charge: 100},
		{name: "no gold", gold: 0},
		{name: "partial balance", gold: 7, charge: 7},
		{name: "daily benefit", gold: 200, free: true},
		{name: "failed insert rolls back charge", gold: 200, charge: 100, fail: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT gold FROM users.*FOR UPDATE").WithArgs("player").
				WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(test.gold))
			expectDailyIdentifyClaim(mock, "player", test.free)
			if test.charge > 0 {
				mock.ExpectExec("UPDATE users SET gold = gold -").WithArgs(test.charge, "player").
					WillReturnResult(sqlmock.NewResult(0, 1))
			}
			insert := mock.ExpectExec("INSERT INTO user_inventory").WithArgs("player", "secret", 80, identifiedGearJSON{})
			if test.fail {
				insert.WillReturnError(errors.New("disk full"))
				mock.ExpectRollback()
			} else {
				insert.WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			}
			_, cost, err := (&Bot{DB: db}).storeAutoIdentifiedDrop(context.Background(), "player", content.Gear{
				ID: "secret", Rarity: content.RarityEternal, MaxDurability: 80, Unidentified: true,
			})
			if (err != nil) != test.fail || (!test.fail && cost != test.charge) {
				t.Fatalf("cost=%d err=%v, want cost=%d failure=%v", cost, err, test.charge, test.fail)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAutoIdentifyExistingItems(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, gear_id, item_data FROM user_inventory.*ORDER BY id FOR UPDATE").WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"id", "gear_id", "item_data"}).
			AddRow(1, "secret", `{"id":"secret","rarity":0,"unidentified":true}`).
			AddRow(2, "secret", `{"id":"secret","rarity":8,"unidentified":true}`))
	mock.ExpectQuery("SELECT slot, gear_id, item_data FROM user_gear.*ORDER BY slot FOR UPDATE").WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"slot", "gear_id", "item_data"}).
			AddRow("Head", "secret", `{"id":"secret","rarity":2,"unidentified":true}`))
	mock.ExpectQuery("SELECT gold FROM users.*FOR UPDATE").WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(200))
	expectDailyIdentifyClaim(mock, "player", false)
	mock.ExpectExec("UPDATE users SET gold = gold -").WithArgs(int64(111), "player").
		WillReturnResult(sqlmock.NewResult(0, 1))
	for _, id := range []int64{1, 2} {
		mock.ExpectExec("UPDATE user_inventory SET item_data=jsonb_set").WithArgs(id, "player").
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectExec("UPDATE user_gear SET item_data=jsonb_set").WithArgs("Head", "player").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	count, cost, err := (&Bot{DB: db}).autoIdentifyItems(context.Background(), "player")
	if err != nil || count != 3 || cost != 111 {
		t.Fatalf("count=%d cost=%d err=%v", count, cost, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
