package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"ts3news/internal/content"
)

func TestHandleAbyssEquipBestLootSynchronizesEquipment(t *testing.T) {
	for _, test := range []struct {
		name       string
		currentSTR int
		lockError  bool
		wantOK     bool
		wantError  string
	}{
		{name: "current stronger equipment rejects stale recommendation", currentSTR: 30, wantError: "no longer the best upgrade"},
		{name: "current weaker equipment allows reservation", currentSTR: 10, wantOK: true},
		{name: "lock failure preserves escrow selection", lockError: true, wantError: "db"},
	} {
		t.Run(test.name, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = database.Close() })
			const uid = "delver"
			expectEquipBestActiveRun(mock, uid)
			gear := content.Gear{ID: "loot-head", Slot: content.SlotHead, Rarity: content.RarityCommon, MaxDurability: 20, Stats: content.Stats{STR: 20}}
			grant := mustAbyssLootGrantJSON(t, abyssLootGrant{Type: "gear", Gear: &gear})
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT id, item_data, equip_on_bank.*FROM abyss_escrow_loot.*FOR UPDATE").WithArgs(uid).
				WillReturnRows(sqlmock.NewRows([]string{"id", "item_data", "equip_on_bank"}).AddRow(31, grant, false))
			lock := mock.ExpectExec("LOCK TABLE user_gear IN SHARE MODE")
			if test.lockError {
				lock.WillReturnError(errors.New("lock unavailable"))
			} else {
				lock.WillReturnResult(sqlmock.NewResult(0, 0))
				current := gear
				current.ID, current.Stats.STR = "equipped-head", test.currentSTR
				data, err := json.Marshal(current)
				if err != nil {
					t.Fatal(err)
				}
				mock.ExpectQuery("SELECT slot, gear_id, item_data, durability FROM user_gear").WithArgs(uid).
					WillReturnRows(sqlmock.NewRows([]string{"slot", "gear_id", "item_data", "durability"}).AddRow(string(current.Slot), current.ID, string(data), 20))
			}
			if test.wantOK {
				mock.ExpectExec("UPDATE abyss_escrow_loot SET equip_on_bank=TRUE").WithArgs(int64(31), uid).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}
			request := httptest.NewRequest("POST", "/api/abyss/loot/equip_best", strings.NewReader(`{"id":31}`))
			recorder := httptest.NewRecorder()
			(&WebServer{bot: &Bot{DB: database}}).handleAbyssEquipBestLoot(recorder, request, uid)
			var response struct {
				OK    bool   `json:"ok"`
				Error string `json:"error"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.OK != test.wantOK || !strings.Contains(response.Error, test.wantError) {
				t.Errorf("response = %+v, want ok=%t and error containing %q", response, test.wantOK, test.wantError)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func expectEquipBestActiveRun(mock sqlmock.Sqlmock, uid string) {
	mock.ExpectQuery("SELECT depth, escrow, tier, insured").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"depth", "escrow", "tier", "insured", "revived", "floor_type", "modifier", "event_state", "started_at", "last_action_at", "coop_uid", "momentum", "bank_locked_floors", "last_stand_used", "revive_locked", "checkpoint_start", "express_until", "comeback", "last_rest_depth"}).
			AddRow(1, 0, "normal", 0, false, "", "", nil, time.Now(), time.Now(), nil, 0, 0, false, false, 0, 0, false, 0))
	for _, query := range []string{
		"SELECT level, prestige FROM users",
		"SELECT title, title_mult, title_expires FROM users",
		"SELECT artifact_mult, artifact_name, artifact_durability FROM users",
		"SELECT slot, gear_id, durability, enchantment_id, item_data FROM user_gear",
		"SELECT abyss_win_streak FROM users",
		"SELECT skill_id FROM user_skills",
		"SELECT ultimate_id, current_cooldown FROM user_ultimate_skills",
		"SELECT cons_id, remaining_fights FROM user_consumables",
		"SELECT node_id FROM user_abyss_tree",
		"SELECT value FROM app_meta",
		"SELECT value FROM app_meta",
	} {
		mock.ExpectQuery(query).WillReturnError(sql.ErrNoRows)
	}
	mock.ExpectQuery("SELECT current_hp, level FROM users").WithArgs(uid).
		WillReturnRows(sqlmock.NewRows([]string{"current_hp", "level"}).AddRow(100, 1))
}
