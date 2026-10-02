package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
	"time"
	"ts3news/internal/content"
	"ts3news/internal/rift"
)

func TestRiftRepeatedFinishNeverDeliversLootAgain(t *testing.T) {
	for _, status := range []string{"banked", "complete"} {
		for _, kind := range []string{"bank", "exit", "next", "advance"} {
			for _, revision := range []int{4, 5, 6} {
				t.Run(fmt.Sprintf("%s-%s-%d", status, kind, revision), func(t *testing.T) {
					db, mock, err := sqlmock.New()
					if err != nil {
						t.Fatal(err)
					}
					defer func() { _ = db.Close() }()
					now := time.Unix(100, 0)
					run := rift.NewRunAtLevel("finished", rift.Build{HP: 300}, now, riftMobCatalog(now), 100)
					run.Room = 2
					run.Status = status
					run.Epoch = "2"
					run.Revision = 5
					run.LastRequestID = "repeat-finish-request"
					run.BankedGold = 30
					run.BankedItems = []string{"Already delivered"}
					run.BankedAtMS = 100000
					run.Drops = []rift.Drop{{ID: "paid", Gold: 30, Collected: true, Banked: true, Gear: &content.Gear{ID: "ABYSS_TEST", Name: "Already delivered", MaxDurability: 80}}}
					raw, err := json.Marshal(run)
					if err != nil {
						t.Fatal(err)
					}
					mock.ExpectBegin()
					mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
					mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
					mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("rift_brawl:owner").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(raw)))
					mock.ExpectRollback()
					out, err := (&Bot{DB: db}).updateRift(context.Background(), "owner", riftRequest{Kind: kind, RunID: run.ID, Revision: revision, RequestID: "repeat-finish-request"}, rift.Build{}, now.Add(time.Second))
					if revision == 6 {
						if !errors.Is(err, errRiftConflict) || out != nil {
							t.Fatal("fresh finish on terminal run accepted")
						}
					} else if err != nil || out == nil || out.BankedGold != 30 || len(out.BankedItems) != 1 || out.BankedAtMS != 100000 || out.Revision != 5 || out.Status != status {
						t.Fatalf("replay changed settled receipt: %v %+v", err, out)
					}
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Fatal(err)
					}
				})
			}
		}
	}
}
