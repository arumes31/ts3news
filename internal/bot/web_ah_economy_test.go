package bot

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAHMaterialOrderDistinguishesUnconfirmedCommit(t *testing.T) {
	t.Parallel()
	for _, stage := range []string{"reserve", "insert", "commit"} {
		t.Run(stage, func(t *testing.T) {
			t.Parallel()
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = db.Close() }()
			failure := errors.New("private database failure")
			mock.ExpectBegin()
			reserve := mock.ExpectExec(`UPDATE users SET gold=gold-$1 WHERE client_uid=$2 AND gold >= $1`).
				WithArgs(int64(500), "buyer")
			if stage == "reserve" {
				reserve.WillReturnError(failure)
			} else {
				reserve.WillReturnResult(sqlmock.NewResult(0, 1))
				insert := mock.ExpectExec(`INSERT INTO abyss_material_orders (buyer_uid,material,unit_price,remaining,escrow_gold)
					VALUES ($1,$2,$3,$4,$5)`).WithArgs("buyer", "dust", int64(250), 2, int64(500))
				if stage == "insert" {
					insert.WillReturnError(failure)
				} else {
					insert.WillReturnResult(sqlmock.NewResult(7, 1))
					mock.ExpectCommit().WillReturnError(failure)
				}
			}
			if stage != "commit" {
				mock.ExpectRollback()
			}
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/ah/material/order", strings.NewReader(`{"material":"dust","count":2,"unit_price":250}`))
			(&WebServer{bot: &Bot{DB: db}}).handleAHMaterialOrder(response, request, "buyer")
			var result struct {
				OK          bool   `json:"ok"`
				Unconfirmed bool   `json:"unconfirmed"`
				Error       string `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatalf("invalid response: %s", response.Body.String())
			}
			if result.OK || result.Unconfirmed != (stage == "commit") || result.Error == "" || strings.Contains(result.Error, failure.Error()) {
				t.Fatalf("%s failure response = %s", stage, response.Body.String())
			}
			if stage == "commit" && !strings.Contains(result.Error, "check your orders") {
				t.Fatalf("unconfirmed response needs recovery instructions: %s", response.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
