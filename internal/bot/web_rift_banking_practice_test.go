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

func TestRiftPracticeBankOnlySavesDemoAndReplaysReceipt(t *testing.T) {
	for _, stage := range []string{"ready", "early", "replay"} {
		t.Run(stage, func(t *testing.T) {
			req := riftRequest{Kind: "practice_bank", RequestID: "practice-bank-request", RunID: "demo", Revision: 5}
			if !validRiftRequest(req) || !validRiftModeAction("banking", req.Kind) || validRiftModeAction("", req.Kind) || validRiftModeAction("skills", req.Kind) {
				t.Fatal("invalid demo scope")
			}
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			run, _ := rift.NewPracticeRun("demo", rift.Build{HP: 100}, "banking", time.Unix(100, 0))
			run.Epoch = "2"
			run.Revision = 4
			if stage != "early" {
				run.Practice.CheckpointReady = true
				for i := range run.Drops {
					run.Drops[i].Collected = true
				}
			}
			if stage == "replay" {
				if err = run.PracticeTool("practice_bank"); err != nil {
					t.Fatal(err)
				}
				req.Revision = 4
				run.LastRequestID = req.RequestID
			}
			saved, _ := json.Marshal(run)
			key := "rift_practice:owner:banking"
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
			mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
			mock.ExpectQuery("SELECT value FROM app_meta").WithArgs(key).WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(saved)))
			if stage != "ready" {
				mock.ExpectRollback()
			} else {
				mock.ExpectExec("INSERT INTO app_meta").WithArgs(key, riftSnapshotCheck(func(result *rift.Run) bool {
					return result.Revision == 5 && result.Practice.Completed && result.Status == "complete" && result.Gold == 0 && result.BankedGold == 0 && len(result.BankedItems) == 0 && len(result.Drops) == 3 && result.Drops[0].Banked && result.Drops[1].Banked && result.Drops[2].Banked
				})).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			}
			result, err := (&Bot{DB: database}).updateRiftMode(context.Background(), "owner", req, rift.Build{}, time.Unix(200, 0), "banking")
			if stage == "early" {
				if !errors.Is(err, errRiftConflict) || result != nil {
					t.Fatalf("early bank: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if stage == "replay" {
				after, _ := json.Marshal(result)
				if string(after) != string(saved) {
					t.Fatal("duplicate changed receipt")
				}
			}
			if err = mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
