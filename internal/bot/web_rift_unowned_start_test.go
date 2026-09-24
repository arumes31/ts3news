package bot

import (
	"github.com/DATA-DOG/go-sqlmock"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRiftUnownedStartSelectionRejectedBeforeRunTransaction(t *testing.T) {
	for _, skills := range []string{`["unowned"]`, `["S_EQ"]`, `["ULT_REVIVAL"]`} {
		t.Run(skills, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			empty := func(query string, columns ...string) {
				mock.ExpectQuery(query).WillReturnRows(sqlmock.NewRows(columns))
			}
			mock.ExpectQuery("SELECT level, prestige").WillReturnRows(sqlmock.NewRows([]string{"level", "prestige"}).AddRow(20, 0))
			mock.ExpectQuery("SELECT title, title_mult, title_expires").WillReturnRows(sqlmock.NewRows([]string{"title", "title_mult", "title_expires"}).AddRow(nil, nil, nil))
			mock.ExpectQuery("SELECT artifact_mult, artifact_name, artifact_durability").WillReturnRows(sqlmock.NewRows([]string{"artifact_mult", "artifact_name", "artifact_durability"}).AddRow(nil, nil, 0))
			empty("SELECT slot, gear_id, durability, enchantment_id, item_data", "slot", "gear_id", "durability", "enchantment_id", "item_data")
			mock.ExpectQuery("SELECT abyss_win_streak").WillReturnRows(sqlmock.NewRows([]string{"abyss_win_streak"}).AddRow(0))
			empty("SELECT skill_id FROM user_skills", "skill_id")
			empty("SELECT ultimate_id, current_cooldown", "ultimate_id", "current_cooldown")
			empty("SELECT cons_id, remaining_fights", "cons_id", "remaining_fights")
			empty("SELECT node_id FROM user_abyss_tree", "node_id")
			empty("SELECT value FROM app_meta", "value")
			mock.ExpectQuery("SELECT nickname, level").WillReturnRows(sqlmock.NewRows([]string{"nickname", "level"}).AddRow("Test", 20))
			empty("SELECT skill_id FROM user_skills", "skill_id")
			empty("SELECT node_id FROM user_abyss_tree", "node_id")
			empty("SELECT slot, gear_id, item_data", "slot", "gear_id", "item_data")
			empty("SELECT p.name,p.mob_type", "name")
			empty("SELECT ultimate_id, current_cooldown", "ultimate_id", "current_cooldown")
			mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("abyss_class_build:owner").WillReturnRows(sqlmock.NewRows([]string{"value"}))
			empty("SELECT ultimate_id FROM user_ultimate_skills", "ultimate_id")
			req := httptest.NewRequest(http.MethodPost, "https://game.test/api/abyss/rift", strings.NewReader(`{"kind":"start","run_id":"existing-run","request_id":"unowned-start-selection","level_id":1,"skills":`+skills+`}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			(&WebServer{bot: &Bot{DB: db}}).handleRiftAPI(w, req, "owner")
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "choose owned skills") {
				t.Fatalf("unexpected rejection: %d %s", w.Code, w.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
