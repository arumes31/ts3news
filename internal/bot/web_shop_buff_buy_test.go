package bot

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestShopBuffBuyPersistsTheExactPromotedItem(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	seed, _ := shopWindow(time.Now())
	buffs := shopBuffState{Rarity: 1000, Quantity: 1000}
	var chosen shopItemView
	for _, item := range personalizedShopStock(seed, "delver", buffs, nil) {
		if item.RarityBoosted {
			chosen = item
			break
		}
	}
	if chosen.ID == "" {
		t.Fatal("no promoted stock to test")
	}
	encoded, err := json.Marshal(chosen.gear)
	if err != nil {
		t.Fatal(err)
	}
	const wallet int64 = 25_000_000
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gold FROM users.*FOR UPDATE").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(wallet))
	mock.ExpectQuery("SELECT value FROM app_meta").WithArgs(shopBuffKey("delver")).WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"rarity":1000,"quantity":1000}`))
	mock.ExpectExec("UPDATE users SET gold = gold -").WithArgs(chosen.Price, "delver", wallet).WillReturnResult(sqlmock.NewResult(0, 1))
	// An identical equipped copy is not an upgrade: deliver this one to inventory.
	mock.ExpectQuery("SELECT gear_id, item_data FROM user_gear").WithArgs("delver", chosen.Slot).WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow(chosen.gear.ID, string(encoded)))
	mock.ExpectExec("INSERT INTO user_inventory").WithArgs("delver", chosen.gear.ID, chosen.gear.MaxDurability, string(encoded)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT gold FROM users").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(wallet - chosen.Price))
	mock.ExpectCommit()
	body, err := json.Marshal(map[string]any{"id": chosen.ID, "expected_gold": wallet, "stock_revision": shopStockRevision(seed, "delver", buffs)})
	if err != nil {
		t.Fatal(err)
	}
	server := &WebServer{bot: &Bot{DB: database}}
	response := httptest.NewRecorder()
	server.handleBuyAPI(response, httptest.NewRequest(http.MethodPost, "/api/shop/buy", strings.NewReader(string(body))), "delver")
	if !strings.Contains(response.Body.String(), `"ok":true`) {
		t.Fatal(response.Body.String())
	}
	loaded, ok := server.bot.makeGear(chosen.gear.ID, sql.NullString{String: string(encoded), Valid: true})
	if !ok || loaded.Rarity != chosen.gear.Rarity || loaded.CombatRating() != chosen.CR {
		t.Fatal("promoted rarity did not survive item_data reload")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestShopBuffBuyRejectsStaleOrForeignStockBeforeDebit(t *testing.T) {
	seed, _ := shopWindow(time.Now())
	state := shopBuffState{Rarity: 10, Quantity: 20}
	oldPricing := sha256.Sum256([]byte(fmt.Sprintf("shop-v1:%d:delver:10:20", seed)))
	currentOffer := personalizedShopStock(seed, "delver", state, nil)[0]
	for _, tc := range []struct{ name, revision, id string }{
		{"previous pricing", fmt.Sprintf("%x", oldPricing[:16]), currentOffer.ID},
		{"missing review", "", "anything"},
		{"previous buffs", shopStockRevision(seed, "delver", shopBuffState{Rarity: 9, Quantity: 20}), "anything"},
		{"previous rotation", shopStockRevision(seed-1, "delver", state), "anything"},
		{"other user", shopStockRevision(seed, "other", state), "anything"},
		{"forged offer", shopStockRevision(seed, "delver", state), "SHOP_BOOST_999_forged"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT gold FROM users.*FOR UPDATE").WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(25_000_000))
			mock.ExpectQuery("SELECT value FROM app_meta").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"rarity":10,"quantity":20}`))
			mock.ExpectRollback()
			body, err := json.Marshal(map[string]any{"id": tc.id, "expected_gold": 25_000_000, "stock_revision": tc.revision})
			if err != nil {
				t.Fatal(err)
			}
			server := &WebServer{bot: &Bot{DB: database}}
			response := httptest.NewRecorder()
			server.handleBuyAPI(response, httptest.NewRequest(http.MethodPost, "/api/shop/buy", strings.NewReader(string(body))), "delver")
			if !strings.Contains(response.Body.String(), `"review_required":true`) {
				t.Fatal(response.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
