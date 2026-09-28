package bot

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"net/http/httptest"
	"testing"
	"time"
	"ts3news/internal/rift"
)

func TestRiftPotionInventoryAndSaveShareTransaction(t *testing.T) {
	for _, mode := range []string{"success", "save_failure", "empty", "replay"} {
		t.Run(mode, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer database.Close()
			now := time.Unix(100, 0)
			run := rift.NewRunAtLevel("potion", rift.Build{HP: 100}, now, riftMobCatalog(now), 1)
			run.Epoch = "2"
			run.Paused = false
			run.Player.HP = 30
			if mode == "replay" {
				if err := run.UseHealingPotion(50); err != nil {
					t.Fatal(err)
				}
				run.Revision = 1
				run.LastRequestID = "potion-request-01"
			}
			raw, err := json.Marshal(run)
			if err != nil {
				t.Fatal(err)
			}
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
			mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
			mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("rift_brawl:owner").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(raw)))
			if mode != "replay" {
				rows := int64(1)
				if mode == "empty" {
					rows = 0
				}
				mock.ExpectExec("UPDATE user_consumables").WithArgs("owner", "small_health_potion").WillReturnResult(sqlmock.NewResult(0, rows))
				if mode != "empty" {
					mock.ExpectExec("DELETE FROM user_consumables").WithArgs("owner", "small_health_potion").WillReturnResult(sqlmock.NewResult(0, 0))
					save := mock.ExpectExec("INSERT INTO app_meta").WithArgs("rift_brawl:owner", riftSnapshotCheck(func(saved *rift.Run) bool {
						return saved.Player.HP == 80 && saved.Stats.PotionsUsed == 1 && saved.Revision == 1 && saved.SkillTimers["healing_potion"] == 8
					}))
					if mode == "save_failure" {
						save.WillReturnError(errors.New("save failed"))
					} else {
						save.WillReturnResult(sqlmock.NewResult(0, 1))
					}
				}
			}
			if mode == "success" {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}
			got, err := (&Bot{DB: database}).updateRift(context.Background(), "owner", riftRequest{Kind: "potion", ConsumableID: "small_health_potion", RunID: run.ID, Revision: 1, RequestID: "potion-request-01"}, run.Build, now)
			wantError := mode == "empty" || mode == "save_failure"
			if (err != nil) != wantError {
				t.Fatalf("result: %v", err)
			}
			if !wantError && (got.Player.HP != 80 || got.Stats.PotionsUsed != 1) {
				t.Fatal("potion replay or effect incorrect")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRiftPotionInventoryReadFiltersOwnedSupportedItems(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectQuery("SELECT cons_id, remaining_fights FROM user_consumables").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"cons_id", "remaining_fights"}).
		AddRow("small_health_potion", 2).AddRow("rejuvenation_potion", 3).AddRow("strength_elixir", 1).AddRow("corrupted_great_health_potion", 1).AddRow("unknown", 1).AddRow("elixir_of_life", 0))
	items, err := (&Bot{DB: database}).riftPotions(context.Background(), "owner")
	if err != nil || len(items) != 2 {
		t.Fatalf("inventory: %+v %v", items, err)
	}
	if items[0].ID != "small_health_potion" || items[0].Count != 2 || items[0].HealHP != 50 || items[0].HealFraction != 0 {
		t.Fatal("fixed potion metadata")
	}
	if items[1].ID != "rejuvenation_potion" || items[1].Count != 3 || items[1].HealFraction != .6 || items[1].HealHP != 0 {
		t.Fatal("fractional potion metadata")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRiftPotionInventoryReadIsSeparateAndPracticeHasNoRealInventory(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	server := &WebServer{bot: &Bot{DB: database}}
	mock.ExpectQuery("SELECT cons_id, remaining_fights FROM user_consumables").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"cons_id", "remaining_fights"}))
	for _, url := range []string{"/api/abyss/rift?inventory=potions", "/api/abyss/rift?inventory=potions&practice=boss"} {
		response := httptest.NewRecorder()
		server.handleRiftAPI(response, httptest.NewRequest("GET", url, nil), "owner")
		if response.Code != 200 || response.Header().Get("Cache-Control") != "no-store, no-cache, must-revalidate" {
			t.Fatalf("read failed: %d", response.Code)
		}
		var data struct {
			OK      bool               `json:"ok"`
			Potions []riftPotionOption `json:"potions"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil || !data.OK || data.Potions == nil || len(data.Potions) != 0 {
			t.Fatalf("invalid empty inventory: %s", response.Body.String())
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
