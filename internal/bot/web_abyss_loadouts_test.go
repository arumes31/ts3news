package bot

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ts3news/internal/content"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssGemPresetApplyRejectsInactiveGearBeforeMutation(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	const uid = "delver"
	gear := content.Gear{
		ID: "inactive-head", Slot: content.SlotHead,
		Gemstones: []string{"Ruby"}, Unidentified: true,
	}
	itemData, err := json.Marshal(gear)
	if err != nil {
		t.Fatal(err)
	}
	preset := `{"1":{"name":"Inactive gems","gems":{"Head":["Sapphire","Ruby"]}}}`

	mock.ExpectQuery("SELECT depth, escrow, tier").
		WithArgs(uid).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT value FROM app_meta WHERE key=\$1`).
		WithArgs(abyssGemPresetKey(uid)).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(preset))
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT gear_id, item_data FROM user_gear WHERE client_uid=\$1 AND slot=\$2`).
		WithArgs(uid, string(content.SlotHead)).
		WillReturnRows(sqlmock.NewRows([]string{"gear_id", "item_data"}).AddRow(gear.ID, string(itemData)))
	mock.ExpectRollback()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/loadout/gems", strings.NewReader(`{"action":"apply","slot":1}`))
	(&WebServer{bot: &Bot{DB: database}}).handleAbyssGemPreset(recorder, request, uid)

	var response struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.OK || !strings.Contains(response.Error, "inactive") {
		t.Fatalf("response = %#v, want inactive-gear rejection", response)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
