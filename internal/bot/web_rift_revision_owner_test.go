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

func TestRiftEqualRevisionRequiresWinningRequest(t *testing.T) {
	for _, winner := range []string{"winning-request-id", ""} {
		for _, requestID := range []string{"winning-request-id", "other-tab-request-id"} {
			t.Run(winner+"/"+requestID, func(t *testing.T) {
				database, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}
				defer database.Close()
				run := rift.NewRun("run", rift.Build{HP: 200}, time.Now())
				run.Epoch = "2"
				run.Revision = 5
				raw, err := json.Marshal(run)
				if err != nil {
					t.Fatal(err)
				}
				var fields map[string]any
				if err = json.Unmarshal(raw, &fields); err != nil {
					t.Fatal(err)
				}
				if winner != "" {
					fields["last_request_id"] = winner
				}
				raw, err = json.Marshal(fields)
				if err != nil {
					t.Fatal(err)
				}
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT client_uid FROM users").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
				mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
				mock.ExpectQuery("SELECT value FROM app_meta").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(raw)))
				mock.ExpectRollback()
				out, err := (&Bot{DB: database}).updateRift(context.Background(), "owner", riftRequest{Kind: "step", RunID: "run", Revision: 5, RequestID: requestID}, rift.Build{}, time.Now())
				if winner != "" && winner == requestID {
					if err != nil || out == nil || out.Revision != 5 {
						t.Fatalf("valid retry failed: %v", err)
					}
				} else if !errors.Is(err, errRiftConflict) || out != nil {
					t.Fatalf("different or unknown winner silently accepted: %v", err)
				}
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
