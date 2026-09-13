package bot

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"ts3news/internal/content"
)

func TestAbyssTalentUpgradeRejectsInvalidStoredLevels(t *testing.T) {
	t.Parallel()
	for _, test := range []struct{ name, raw string }{
		{"malformed", `{"dd_0_0":`},
		{"empty", ``},
		{"null object", `null`},
		{"null level", `{"dd_0_0":null}`},
		{"negative requested level", `{"dd_0_0":-2}`},
		{"negative other level", `{"dd_0_0":0,"removed_node":-1}`},
		{"fraction", `{"dd_0_0":1.5}`},
		{"wrong type", `{"dd_0_0":"2"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("player").
				WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("player"))
			mock.ExpectQuery("SELECT value FROM app_meta").WithArgs(abyssTalentKey("player")).
				WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(test.raw))
			mock.ExpectRollback()
			s := &WebServer{bot: &Bot{DB: database}}
			response := httptest.NewRecorder()
			s.handleAbyssTalentUpgrade(response, "player", content.Talent{Key: "dd_0_0"})
			var result struct {
				OK    bool   `json:"ok"`
				Error string `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.OK || result.Error != "invalid talent data" {
				t.Fatalf("response = %s; want invalid talent data", response.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAbyssTalentPersistenceRejectsNonpositiveCost(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		cost int64
	}{
		{"negative", -10}, {"zero", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			// Invalid prices must be rejected before accessing the transaction.
			spent, err := persistAbyssTalentUpgrade(nil, "player", test.cost, map[string]int{"dd_0_0": 1})
			if err == nil || spent {
				t.Fatalf("spent=%t, err=%v; want rejected cost", spent, err)
			}
		})
	}
}

func TestAbyssTalentUpgradeAcceptsValidStoredLevels(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, raw, saved string
		missing          bool
		cost             int64
	}{
		{name: "first allocation", missing: true, saved: `{"dd_0_0":1}`, cost: 10},
		{name: "empty allocation", raw: `{}`, saved: `{"dd_0_0":1}`, cost: 10},
		{name: "existing allocation", raw: `{"dd_0_0":2,"removed_node":4}`, saved: `{"dd_0_0":3,"removed_node":4}`, cost: 30},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("player").
				WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("player"))
			query := mock.ExpectQuery("SELECT value FROM app_meta").WithArgs(abyssTalentKey("player"))
			if test.missing {
				query.WillReturnError(sql.ErrNoRows)
			} else {
				query.WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(test.raw))
			}
			mock.ExpectExec("UPDATE users SET abyss_tokens").WithArgs(test.cost, "player").
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectExec("INSERT INTO app_meta").WithArgs(abyssTalentKey("player"), test.saved).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
			mock.ExpectQuery("SELECT abyss_tokens FROM users").WithArgs("player").
				WillReturnRows(sqlmock.NewRows([]string{"abyss_tokens"}).AddRow(20))
			mock.ExpectQuery("SELECT abyss_talent_credit FROM users").WithArgs("player").
				WillReturnRows(sqlmock.NewRows([]string{"abyss_talent_credit"}).AddRow(0))
			s := &WebServer{bot: &Bot{DB: database}}
			response := httptest.NewRecorder()
			s.handleAbyssTalentUpgrade(response, "player", content.Talent{Key: "dd_0_0"})
			var result struct {
				OK    bool `json:"ok"`
				Level int  `json:"level"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if !result.OK || int64(result.Level) != test.cost/10 {
				t.Fatalf("response = %s; want allocated level %d", response.Body.String(), test.cost/10)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
