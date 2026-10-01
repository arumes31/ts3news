package bot

import (
	"encoding/json"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"ts3news/internal/rift"
)

func TestRiftEconomyResetRejectsActiveActionsAndReplays(t *testing.T) {
	for _, kind := range []string{"step", "pause", "resume", "bank", "exit", "next", "advance"} {
		for _, revision := range []int{4, 5} {
			t.Run(fmt.Sprintf("%s/%d", kind, revision), func(t *testing.T) {
				database, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = database.Close() }()
				run := rift.NewRun("old-economy-run", rift.Build{HP: 200}, time.Now())
				run.Epoch = "old"
				run.Revision = 4
				run.Gold = 99
				run.Drops = []rift.Drop{{ID: "old-drop", Gold: 99, Collected: true}}
				if kind == "bank" || kind == "exit" || kind == "next" || kind == "advance" {
					run.Status = "cleared"
				}
				raw, err := json.Marshal(run)
				if err != nil {
					t.Fatal(err)
				}
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT client_uid FROM users").WithArgs("owner").WillReturnRows(sqlmock.NewRows([]string{"client_uid"}).AddRow("owner"))
				mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("new"))
				mock.ExpectQuery("SELECT value FROM app_meta").WithArgs("rift_brawl:owner").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(raw)))
				mock.ExpectRollback()
				body := fmt.Sprintf(`{"kind":"%s","run_id":"old-economy-run","request_id":"economy-reset-action","revision":%d}`, kind, revision)
				request := httptest.NewRequest(http.MethodPost, "https://game.test/api/abyss/rift", strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				response := httptest.NewRecorder()
				(&WebServer{bot: &Bot{DB: database}}).handleRiftAPI(response, request, "owner")
				if response.Code != 409 || !strings.Contains(response.Body.String(), `"code":"ECONOMY_RESET"`) {
					t.Fatalf("%d: %s", response.Code, response.Body.String())
				}
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestRiftEconomyExpirationKeepsHistoryOnce(t *testing.T) {
	run := rift.NewRun("old", rift.Build{HP: 200}, time.Now())
	run.Gold = 99
	run.Drops = []rift.Drop{{ID: "old-loot", Gold: 99, Collected: true}}
	run.BankedGold = 80
	run.BankedObjectiveGold = 20
	run.BankedItems = []string{"Recent sword"}
	run.BankedItemsTotal = 1234
	run.PastExpeditions = rift.CareerTotals{Gold: 20, Gear: 2}
	run.CompletedLevels = []int{1, 7}
	for range 3 {
		expireRiftEconomy(run)
		if run.Status != "expired" || run.Gold != 0 || len(run.Drops) != 0 || run.BankedGold != 0 || run.BankedObjectiveGold != 0 || run.TotalBankedItems() != 0 {
			t.Fatal("expired rewards survived")
		}
		if totals := run.RecordedTotals(); totals.Gold != 100 || totals.Gear != 1236 {
			t.Fatalf("history repeated or lost: %+v", totals)
		}
		if len(run.CompletedLevels) != 2 {
			t.Fatal("completion records lost")
		}
	}
	fresh := rift.NewRunAtLevel("new", rift.Build{HP: 250}, time.Now(), riftMobCatalog(time.Now()), 1)
	fresh.InheritCampaignHistory(run)
	if fresh.Gold != 0 || fresh.BankedGold != 0 || fresh.TotalBankedItems() != 0 || fresh.RecordedTotals().Gold != 100 || fresh.RecordedTotals().Gear != 1236 || len(fresh.CompletedLevels) != 2 {
		t.Fatal("restart resurrected rewards or lost history")
	}
}
