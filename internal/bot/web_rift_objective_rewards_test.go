package bot

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"ts3news/internal/rift"
)

func TestRiftObjectiveRewardsAtomicAndReplaySafe(t *testing.T) {
	for _, scenario := range []string{"bank", "replay", "bonus failure", "snapshot failure"} {
		t.Run(scenario, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer database.Close()
			bonusFailure := errors.New("bonus write failed")
			snapshotFailure := errors.New("snapshot write failed")
			run := rift.NewRunAtLevel("bonus-run", rift.Build{Name: "Delver", HP: 200}, time.Unix(100, 0), riftMobCatalog(time.Unix(100, 0)), 1)
			run.Room = 2
			run.Status = "cleared"
			run.Epoch = "2"
			run.Revision = 4
			run.Drops = []rift.Drop{{ID: "fight", Gold: 30, Collected: true}}
			run.Gold = 30
			run.Objectives = &rift.MissionObjectives{Mission: 1, Finished: true, RewardPerObjective: 5, Entries: []rift.ObjectiveProgress{{ID: "timed", Status: "complete"}, {ID: "guard", Status: "complete"}, {ID: "no_damage", Status: "failed"}}}
			request := riftRequest{Kind: "bank", RunID: run.ID, Revision: 5, RequestID: "bonus-request"}
			if scenario == "replay" {
				run.Revision = 5
				run.Status = "complete"
				run.Gold = 0
				run.Drops[0].Banked = true
				run.BankedGold = 40
				run.BankedObjectiveGold = 10
				run.Objectives.RewardGold = 10
				run.Objectives.Banked = true
			}
			data, err := json.Marshal(run)
			if err != nil {
				t.Fatal(err)
			}
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
			mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
			mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("rift_brawl:owner").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(data)))
			if scenario == "replay" {
				mock.ExpectRollback()
			} else {
				mock.ExpectExec("SELECT set_config").WithArgs("rift_brawl", request.RequestID, run.ID, "").WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec("UPDATE users SET gold").WithArgs(int64(30), "owner").WillReturnResult(sqlmock.NewResult(0, 1))
				bonus := mock.ExpectExec("UPDATE users SET gold").WithArgs(int64(10), "owner")
				if scenario == "bonus failure" {
					bonus.WillReturnError(bonusFailure)
					mock.ExpectRollback()
				} else {
					bonus.WillReturnResult(sqlmock.NewResult(0, 1))
					snapshot := riftSnapshotCheck(func(saved *rift.Run) bool {
						return saved.BankedGold == 40 && saved.BankedObjectiveGold == 10 && saved.Objectives.RewardGold == 10 && saved.Objectives.Banked && saved.Drops[0].Gold == 30 && saved.Drops[0].Banked && saved.Revision == 5
					})
					write := mock.ExpectExec("INSERT INTO app_meta").WithArgs("rift_brawl:owner", snapshot)
					if scenario == "snapshot failure" {
						write.WillReturnError(snapshotFailure)
						mock.ExpectRollback()
					} else {
						write.WillReturnResult(sqlmock.NewResult(0, 1))
						mock.ExpectCommit()
					}
				}
			}
			out, err := (&Bot{DB: database}).updateRift(context.Background(), "owner", request, rift.Build{}, time.Unix(101, 0))
			if scenario == "bank" || scenario == "replay" {
				if err != nil || out == nil || out.BankedGold != 40 || out.BankedObjectiveGold != 10 || out.Objectives.RewardGold != 10 {
					t.Fatalf("invalid bonus receipt: %v %+v", err, out)
				}
			} else if err == nil || out != nil {
				t.Fatal("failed transaction exposed uncommitted rewards")
			}
			if scenario == "snapshot failure" && !errors.Is(err, snapshotFailure) {
				t.Fatalf("did not reach failing snapshot write: %v", err)
			}
			if scenario == "bonus failure" && !errors.Is(err, bonusFailure) {
				t.Fatalf("did not reach failing bonus write: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
