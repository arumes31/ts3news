package bot

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssPetFeedConsumesOwnedConsumableAtomically(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	const uid = "feeder"
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT client_uid FROM users").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow(uid))
	mock.ExpectQuery("SELECT name,level,hp,max_hp,loyalty,autoskills::text FROM user_pets").
		WithArgs(int64(4), uid).WillReturnRows(sqlmock.NewRows([]string{"name", "level", "hp", "max_hp", "loyalty", "autoskills"}).
		AddRow("Moss", 2, 20, 50, 80, `{}`))
	mock.ExpectQuery("SELECT remaining_fights FROM user_consumables").WithArgs(uid, "small_health_potion").
		WillReturnRows(sqlmock.NewRows([]string{"remaining_fights"}).AddRow(1))
	mock.ExpectExec("DELETE FROM user_consumables").WithArgs(uid, "small_health_potion").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE user_pets SET level=").WithArgs(2, 50, 90, sqlmock.AnyArg(), int64(4), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT cons_id, remaining_fights FROM user_consumables").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"cons_id", "remaining_fights"}))

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/social/pet/feed", strings.NewReader(`{"pet_id":4,"cons_id":"small_health_potion"}`))
	response := httptest.NewRecorder()
	server.handleAbyssPetFeed(response, request, uid)
	if body := response.Body.String(); !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"consumables"`) {
		t.Fatalf("feed response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssPetFusionRejectsFavoriteDonorWithoutMutation(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	const uid = "fusion"
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT pet_id,name,mob_type").WithArgs(uid, int64(1), int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "name", "mob_type", "level", "hp", "max_hp", "str", "def", "spd", "loyalty", "active_slot", "autoskills"}).
			AddRow(1, "Keep", "Elite", 5, 50, 50, 20, 20, 20, 80, 0, `{}`).
			AddRow(2, "Star", "Elite", 5, 50, 50, 20, 20, 20, 80, 0, `{"favorite":true}`))
	mock.ExpectRollback()

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/social/pet/fusion", strings.NewReader(`{"keep_pet_id":1,"donor_pet_id":2}`))
	response := httptest.NewRecorder()
	server.handleAbyssPetFusion(response, request, uid)
	if body := response.Body.String(); !strings.Contains(body, "unfavorited reserve") {
		t.Fatalf("fusion response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssPetActivityPersistsDaycareAssignment(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	const uid = "daycare"
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT name,level,hp,max_hp,active_slot,autoskills::text").WithArgs(int64(8), uid).
		WillReturnRows(sqlmock.NewRows([]string{"name", "level", "hp", "max_hp", "active_slot", "autoskills"}).
			AddRow("Pebble", 3, 60, 60, 0, `{}`))
	mock.ExpectExec("UPDATE user_pets SET level=").WithArgs(3, 60, 60, sqlmock.AnyArg(), int64(8), uid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/social/pet/activity", strings.NewReader(`{"pet_id":8,"action":"daycare_start"}`))
	response := httptest.NewRecorder()
	server.handleAbyssPetActivity(response, request, uid)
	if body := response.Body.String(); !strings.Contains(body, `"ok":true`) || !strings.Contains(body, "entered daycare") {
		t.Fatalf("activity response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssPetGiftClaimRejectsFullStableBeforeTransfer(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	const uid = "recipient"
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT client_uid FROM users").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow(uid))
	mock.ExpectQuery("SELECT g.pet_id,g.sender_uid,p.name,p.autoskills::text").WithArgs("GIFT42", uid).
		WillReturnRows(sqlmock.NewRows([]string{"pet_id", "sender_uid", "name", "autoskills"}).AddRow(9, "sender", "Comet", `{}`))
	mock.ExpectQuery("SELECT node_id FROM user_abyss_tree").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"node_id"}))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM user_pets WHERE client_uid=$1")).WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectRollback()

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/social/pet/gift/claim", strings.NewReader(`{"code":"gift42"}`))
	response := httptest.NewRecorder()
	server.handleAbyssPetGiftClaim(response, request, uid)
	if body := response.Body.String(); !strings.Contains(body, "make room") {
		t.Fatalf("gift claim response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
