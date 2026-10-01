package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
	"time"
	"ts3news/internal/rift"
)

func TestRiftDuplicateAdvanceCannotSkipLaterCheckpoint(t *testing.T) {
	for _, status := range []string{"fighting", "cleared"} {
		for _, revision := range []int{5, 9} {
			t.Run(fmt.Sprintf("%s-revision%d", status, revision), func(t *testing.T) {
				db, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = db.Close() }()
				now := time.Unix(100, 0)
				catalog := riftMobCatalog(now)
				run := rift.NewRunAtLevel("same-expedition", rift.Build{HP: 300}, now, catalog, 10)
				run.Status = "cleared"
				run.Room = 2
				run.FinishCheckpoint("advance", catalog)
				run.Status = status
				run.Room = 2
				run.Revision = 9
				run.LastRequestID = "last-room-step-request"
				run.Epoch = "2"
				run.Drops = []rift.Drop{{ID: "new-room-loot", Gold: 70, Collected: true}}
				run.BankedGold = 30
				run.BankedItems = []string{"Previous reward"}
				raw, err := json.Marshal(run)
				if err != nil {
					t.Fatal(err)
				}
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
				mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
				mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("rift_brawl:owner").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(raw)))
				mock.ExpectRollback()
				out, err := (&Bot{DB: db}).updateRift(context.Background(), "owner", riftRequest{Kind: "advance", RunID: run.ID, Revision: revision, RequestID: "replayed-advance-request"}, rift.Build{}, now.Add(time.Second))
				if revision == 9 {
					if !errors.Is(err, errRiftConflict) || out != nil { t.Fatal("different request took the current revision") }
					if err := mock.ExpectationsWereMet(); err != nil { t.Fatal(err) }
					return
				}
				if err != nil || out == nil {
					t.Fatalf("duplicate should return confirmed snapshot: %v", err)
				}
				if out.Level.ID != 11 || out.Room != 2 || out.Status != status || out.Revision != 9 || out.BankedGold != 30 || len(out.BankedItems) != 1 || out.Drops[0].Banked || out.History[10].Completions != 1 || out.History[11].Completions != 0 {
					t.Fatal("duplicate advance changed progress or rewards")
				}
				if len(out.CompletedLevels) != 1 || out.CompletedLevels[0] != 10 {
					t.Fatal("duplicate completed another mission")
				}
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
