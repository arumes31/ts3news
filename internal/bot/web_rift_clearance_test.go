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

func TestRiftFightingCheckpointRequestsNeverReachSettlement(t *testing.T) {
	for _, kind := range []string{"bank", "exit", "next", "advance"} {
		for room := range rift.Rooms {
			for _, paused := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s-room%d-paused%t", kind, room, paused), func(t *testing.T) {
					db, mock, err := sqlmock.New()
					if err != nil {
						t.Fatal(err)
					}
					defer func() { _ = db.Close() }()
					now := time.Unix(100, 0)
					run := rift.NewRunAtLevel("active", rift.Build{HP: 300}, now, riftMobCatalog(now), 100)
					run.Room = room
					run.Epoch = "2"
					run.Revision = 4
					run.SetPaused(paused, now)
					run.Drops = []rift.Drop{{ID: "unbanked", Gold: 30, Collected: true}}
					raw, err := json.Marshal(run)
					if err != nil {
						t.Fatal(err)
					}
					mock.ExpectBegin()
					mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
					mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
					mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("rift_brawl:owner").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(raw)))
					mock.ExpectRollback()
					out, err := (&Bot{DB: db}).updateRift(context.Background(), "owner", riftRequest{Kind: kind, RunID: run.ID, Revision: 5, RequestID: "premature"}, rift.Build{}, now.Add(time.Second))
					if !errors.Is(err, errRiftConflict) || out != nil {
						t.Fatalf("premature checkpoint returned %v %+v", err, out)
					}
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Fatal(err)
					}
				})
			}
		}
	}
}
