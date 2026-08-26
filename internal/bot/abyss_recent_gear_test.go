package bot

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNormalizeAbyssRecentGearKeepsLatestWithinTwentyFloors(t *testing.T) {
	entries := []abyssRecentGearEntry{
		{GearID: "expired", FloorOrdinal: 9},
		{GearID: "boundary", FloorOrdinal: 10},
		{GearID: "repeat", FloorOrdinal: 15},
		{GearID: "repeat", FloorOrdinal: 29},
		{GearID: "future", FloorOrdinal: 31},
		{GearID: "", FloorOrdinal: 30},
	}
	got := normalizeAbyssRecentGear(entries, 30)
	if len(got) != 2 || got[0].GearID != "boundary" || got[1] != (abyssRecentGearEntry{GearID: "repeat", FloorOrdinal: 29}) {
		t.Fatalf("normalized history = %#v", got)
	}
	ids := abyssRecentGearIDSet(got)
	if !ids["boundary"] || !ids["repeat"] || ids["expired"] || ids["future"] {
		t.Fatalf("protected IDs = %#v", ids)
	}
}

func TestAbyssRecentGearProtectionLoadsCurrentFloorWindow(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectQuery("SELECT abyss_lifetime_floors FROM users").WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"abyss_lifetime_floors"}).AddRow(40))
	mock.ExpectQuery("SELECT value FROM app_meta").WithArgs(abyssRecentGearKey("player")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`[{"gear_id":"old","floor_ordinal":20},{"gear_id":"guarded","floor_ordinal":21}]`))
	ids, floor := (&Bot{DB: database}).abyssRecentGearProtection("player")
	if floor != 41 || len(ids) != 1 || !ids["guarded"] {
		t.Fatalf("floor = %d, IDs = %#v", floor, ids)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecordAbyssRecentGearDropReplacesAndPrunesHistory(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectQuery("SELECT abyss_lifetime_floors FROM users").WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"abyss_lifetime_floors"}).AddRow(30))
	mock.ExpectQuery("SELECT value FROM app_meta.*FOR UPDATE").WithArgs(abyssRecentGearKey("player")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`[{"gear_id":"expired","floor_ordinal":10},{"gear_id":"same","floor_ordinal":20}]`))
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs(abyssRecentGearKey("player"), `[{"gear_id":"same","floor_ordinal":31}]`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	if err := recordAbyssRecentGearDrop(database, "player", "same"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecordAbyssRecentGearDropRejectsMissingPlayer(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectQuery("SELECT abyss_lifetime_floors FROM users").WithArgs("missing").WillReturnError(sql.ErrNoRows)
	if err := recordAbyssRecentGearDrop(database, "missing", "gear"); err == nil {
		t.Fatal("missing player history update succeeded")
	}
}
