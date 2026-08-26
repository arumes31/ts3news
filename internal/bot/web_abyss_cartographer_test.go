package bot

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSaveAbyssCartographerRouteInTxPersistsRouteAndEventAnchor(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	route := abyssCartographerRoute{
		Floors: []abyssCartographerFloor{
			{Depth: 21, Type: "combat"},
			{Depth: 22, Type: "event"},
		},
		NextEventDepth: 25,
	}
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs(
			abyssCartographerRouteKey("delver"),
			`{"floors":[{"depth":21,"type":"combat"},{"depth":22,"type":"event"}],"next_event_depth":25}`,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs(abyssNextEventDepthKey("delver"), "25").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := database.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := saveAbyssCartographerRouteInTx(
		context.Background(),
		tx,
		"delver",
		route,
	); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdvanceAbyssCartographerRoutePersistsRemainingFloors(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM app_meta WHERE key=$1 FOR UPDATE")).
		WithArgs(abyssCartographerRouteKey("delver")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(
			`{"floors":[{"depth":21,"type":"combat"},{"depth":22,"type":"event"}],"next_event_depth":25}`,
		))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE app_meta SET value=$1 WHERE key=$2")).
		WithArgs(
			`{"floors":[{"depth":22,"type":"event"}],"next_event_depth":25}`,
			abyssCartographerRouteKey("delver"),
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs(abyssNextEventDepthKey("delver"), "25").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	view := (&Bot{DB: database}).advanceAbyssCartographerRoute("delver", 21)
	if !view.Active || view.Remaining != 1 || view.Floors[0].Depth != 22 {
		t.Fatalf("view = %+v", view)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdvanceAbyssCartographerRouteDeletesCompletedChart(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM app_meta WHERE key=$1 FOR UPDATE")).
		WithArgs(abyssCartographerRouteKey("delver")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(
			`{"floors":[{"depth":22,"type":"event"}],"next_event_depth":25}`,
		))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM app_meta WHERE key=$1")).
		WithArgs(abyssCartographerRouteKey("delver")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs(abyssNextEventDepthKey("delver"), "25").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	view := (&Bot{DB: database}).advanceAbyssCartographerRoute("delver", 22)
	if view.Active || view.Remaining != 0 || len(view.Floors) != 0 {
		t.Fatalf("view = %+v", view)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestClearAbyssCartographerForecastInTxIsRunScoped(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM app_meta WHERE key IN").
		WithArgs(
			abyssCartographerRouteKey("delver"),
			abyssNextEventDepthKey("delver"),
			abyssEventPreviewKey("delver"),
		).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	tx, err := database.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := clearAbyssCartographerForecastInTx(
		context.Background(),
		tx,
		"delver",
	); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
