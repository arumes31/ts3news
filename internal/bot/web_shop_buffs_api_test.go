package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestShopBuffPurchaseAtomicAndIndependent(t *testing.T) {
	for _, tc := range []struct {
		name, kind, state, next string
		owned, price            int64
	}{
		{"first rarity", "rarity", `{"rarity":0,"quantity":8}`, `{"rarity":1,"quantity":8}`, 0, 1_000_000},
		{"second quantity", "quantity", `{"rarity":42,"quantity":1}`, `{"rarity":42,"quantity":2}`, 1, 2_000_000},
		{"cap remains", "quantity", `{"rarity":3,"quantity":1000}`, `{"rarity":3,"quantity":1001}`, 1000, 1_000_000_000},
		{"rarity beyond 100 percent", "rarity", `{"rarity":1000,"quantity":3}`, `{"rarity":1001,"quantity":3}`, 1000, 1_000_000_000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			gold := int64(2_000_000_000)
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT gold FROM users.*FOR UPDATE").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(gold))
			mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("shop_permanent_buffs_delver").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(tc.state))
			mock.ExpectExec("UPDATE users SET gold = gold -").WithArgs(tc.price, "delver").WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectExec("INSERT INTO app_meta").WithArgs("shop_permanent_buffs_delver", tc.next).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
			body, err := json.Marshal(map[string]any{"kind": tc.kind, "expected_owned": tc.owned, "expected_gold": gold})
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/api/shop/buffs", strings.NewReader(string(body)))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			server := &WebServer{bot: &Bot{DB: database}}
			server.handleShopBuffAPI(response, request, "delver")
			var result struct {
				OK    bool          `json:"ok"`
				Gold  int64         `json:"gold"`
				Buffs shopBuffState `json:"buffs"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if !result.OK || result.Gold != gold-tc.price {
				t.Fatalf("purchase = %s", response.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestShopBuffPurchaseRejectsUnsafeOrStaleRequests(t *testing.T) {
	for _, tc := range []struct {
		name, body, contentType, origin, stored string
		gold                                    int64
		reads                                   bool
	}{
		{"unknown kind", `{"kind":"gold","expected_owned":0,"expected_gold":1}`, "application/json", "", "", 1, false},
		{"missing review", `{"kind":"rarity"}`, "application/json", "", "", 1, false},
		{"cross origin", `{"kind":"rarity","expected_owned":0,"expected_gold":1}`, "application/json", "https://attacker.invalid", "", 1, false},
		{"form post", `{"kind":"rarity","expected_owned":0,"expected_gold":1}`, "text/plain", "", "", 1, false},
		{"insufficient gold", `{"kind":"rarity","expected_owned":0,"expected_gold":999999}`, "application/json", "", `{"rarity":0,"quantity":0}`, 999999, true},
		{"changed wallet", `{"kind":"rarity","expected_owned":0,"expected_gold":2000000}`, "application/json", "", `{"rarity":0,"quantity":0}`, 1000000, true},
		{"replayed capped price", `{"kind":"rarity","expected_owned":1000,"expected_gold":2000000000}`, "application/json", "", `{"rarity":1001,"quantity":0}`, 2000000000, true},
		{"corrupt state", `{"kind":"rarity","expected_owned":0,"expected_gold":2000000}`, "application/json", "", `{"rarity":-1,"quantity":0}`, 2000000, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			if tc.reads {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT gold FROM users.*FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(tc.gold))
				mock.ExpectQuery("SELECT value FROM app_meta").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(tc.stored))
				mock.ExpectRollback()
			}
			request := httptest.NewRequest(http.MethodPost, "/api/shop/buffs", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", tc.contentType)
			if tc.origin != "" {
				request.Header.Set("Origin", tc.origin)
			}
			response := httptest.NewRecorder()
			server := &WebServer{bot: &Bot{DB: database}}
			server.handleShopBuffAPI(response, request, "delver")
			if !strings.Contains(response.Body.String(), `"ok":false`) {
				t.Fatalf("unsafe purchase = %s", response.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestShopBuffPurchaseFailureDoesNotInviteDuplicateCharge(t *testing.T) {
	for _, at := range []string{"save", "commit"} {
		t.Run(at, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT gold FROM users.*FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(2000000))
			mock.ExpectQuery("SELECT value FROM app_meta").WillReturnError(sql.ErrNoRows)
			mock.ExpectExec("UPDATE users SET gold = gold -").WillReturnResult(sqlmock.NewResult(0, 1))
			if at == "save" {
				mock.ExpectExec("INSERT INTO app_meta").WillReturnError(errors.New("save failed"))
				mock.ExpectRollback()
			} else {
				mock.ExpectExec("INSERT INTO app_meta").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errors.New("connection lost"))
			}
			request := httptest.NewRequest(http.MethodPost, "/api/shop/buffs", strings.NewReader(`{"kind":"rarity","expected_owned":0,"expected_gold":2000000}`))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			server := &WebServer{bot: &Bot{DB: database}}
			server.handleShopBuffAPI(response, request, "delver")
			if strings.Contains(response.Body.String(), `"unconfirmed":true`) != (at == "commit") {
				t.Fatalf("incorrect recovery = %s", response.Body.String())
			}
			if strings.Contains(response.Body.String(), `"ok":true`) {
				t.Fatal(response.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
