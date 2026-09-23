package bot

import (
	"context"
	"encoding/json"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
	"time"
	"ts3news/internal/rift"
)

func TestRiftBossRetryStorageAndRequestReplay(t *testing.T) {
	for _, replay := range []bool{false, true} {
		database, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		now := time.Unix(100, 0)
		run := rift.NewRunAtLevel("retry", rift.Build{HP: 100}, now, riftMobCatalog(now), 1)
		run.Room = 2
		run.Status = "defeated"
		run.Epoch = "2"
		run.BankedGold = 50
		if replay {
			if err := run.RetryBossEncounter(now); err != nil {
				t.Fatal(err)
			}
			run.Revision = 1
		}
		data, err := json.Marshal(run)
		if err != nil {
			t.Fatal(err)
		}
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
		mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
		mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("rift_brawl:owner").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(data)))
		if replay {
			mock.ExpectRollback()
		} else {
			mock.ExpectExec("INSERT INTO app_meta").WithArgs("rift_brawl:owner", riftSnapshotCheck(func(saved *rift.Run) bool {
				return saved.Status == "fighting" && saved.Paused && saved.Room == 2 && saved.BankedGold == 50 && saved.Revision == 1
			})).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
		}
		got, err := (&Bot{DB: database}).updateRift(context.Background(), "owner", riftRequest{Kind: "retry_boss", RunID: run.ID, Revision: 1, RequestID: "retry-boss-request"}, run.Build, now)
		if err != nil || got.BankedGold != 50 || got.Status != "fighting" {
			t.Fatalf("retry failed: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
		database.Close()
	}
}
