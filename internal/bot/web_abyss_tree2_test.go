package bot

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestChargeTreeRespecHonorsConfirmedPrice(t *testing.T) {
	t.Parallel()
	week := abyssCurrentWeek(time.Now())
	for _, tc := range []struct {
		name      string
		stored    string
		limit     int
		queryFail bool
		wantOK    bool
		wantFree  bool
	}{
		{name: "free quote became paid", stored: week, limit: 0},
		{name: "paid price exceeds quote", stored: week, limit: abyssTreeRespecTokens - 1},
		{name: "negative ceiling", limit: -1},
		{name: "price lookup failed", limit: 0, queryFail: true},
		{name: "free respec remains free", limit: 0, wantOK: true, wantFree: true},
		{name: "confirmed paid price", stored: week, limit: abyssTreeRespecTokens, wantOK: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = db.Close() }()
			mock.ExpectBegin()
			query := mock.ExpectQuery("SELECT value FROM app_meta").WithArgs(abyssFreeRespecKey("delver"))
			if tc.queryFail {
				query.WillReturnError(errors.New("unavailable"))
			} else {
				query.WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(tc.stored))
			}
			if tc.wantFree {
				mock.ExpectExec("INSERT INTO app_meta").WithArgs(abyssFreeRespecKey("delver"), week).
					WillReturnResult(sqlmock.NewResult(0, 1))
			} else if tc.wantOK {
				mock.ExpectExec("UPDATE users SET abyss_tokens").WithArgs(int64(abyssTreeRespecTokens), "delver").
					WillReturnResult(sqlmock.NewResult(0, 1))
			}
			mock.ExpectRollback()
			tx, err := db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			recorder := httptest.NewRecorder()
			free, ok := chargeTreeRespecQuoted(recorder, tx, "delver", &tc.limit)
			if free != tc.wantFree || ok != tc.wantOK {
				t.Fatalf("free=%v ok=%v, want free=%v ok=%v; response=%s", free, ok, tc.wantFree, tc.wantOK, recorder.Body.String())
			}
			if !ok && !strings.Contains(recorder.Body.String(), `"ok":false`) {
				t.Fatalf("failed quote has no error response: %s", recorder.Body.String())
			}
			if err := tx.Rollback(); err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
