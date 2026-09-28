package bot

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
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
