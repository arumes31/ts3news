//go:build integration

package bot

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/lib/pq"
	"ts3news/internal/rift"
)

func TestRiftActualPotionCommitUncertainty(t *testing.T) {
	database, dsn := riftBankIntegrationDatabase(t)
	for _, committed := range []bool{false, true} {
		t.Run(fmt.Sprintf("committed=%t", committed), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			uid := fmt.Sprintf("potion-commit-%t", committed)
			if _, err := database.ExecContext(ctx, "INSERT INTO users(client_uid,nickname) VALUES($1,'Potion test')", uid); err != nil {
				t.Fatal(err)
			}
			if _, err := database.ExecContext(ctx, "INSERT INTO user_consumables(client_uid,cons_id,remaining_fights) VALUES($1,'small_health_potion',2)", uid); err != nil {
				t.Fatal(err)
			}
			now := time.Now()
			run := rift.NewRunAtLevel("run-"+uid, rift.Build{Name: "Potion test", HP: 100}, now, riftMobCatalog(now), 1)
			run.Epoch, run.Player.HP = "2", 30
			raw, err := encodeRift(run)
			if err != nil {
				t.Fatal(err)
			}
			key := "rift_brawl:" + uid
			if _, err = database.ExecContext(ctx, "INSERT INTO app_meta(key,value) VALUES($1,$2)", key, raw); err != nil {
				t.Fatal(err)
			}
			connector, err := pq.NewConnector(dsn)
			if err != nil {
				t.Fatal(err)
			}
			fault := &riftLostCommitConnector{Connector: connector, committed: committed}
			uncertain := sql.OpenDB(fault)
			defer uncertain.Close()
			bot := &Bot{DB: uncertain}
			request := riftRequest{Kind: "potion", ConsumableID: "small_health_potion", RunID: run.ID, Revision: 1, RequestID: "potion-request-" + uid}
			out, err := bot.updateRift(ctx, uid, request, run.Build, now)
			if out != nil || !errors.Is(err, driver.ErrBadConn) || !fault.fired.Load() {
				t.Fatalf("expected unknown commit outcome: %v", err)
			}
			assertState := func(used bool) {
				t.Helper()
				var count int
				if err := database.QueryRowContext(ctx, "SELECT remaining_fights FROM user_consumables WHERE client_uid=$1 AND cons_id='small_health_potion'", uid).Scan(&count); err != nil {
					t.Fatal(err)
				}
				saved, err := loadRift(ctx, database, uid)
				if err != nil {
					t.Fatal(err)
				}
				if !used {
					if count != 2 || saved.Player.HP != 30 || saved.Revision != 0 || saved.Stats.PotionsUsed != 0 || saved.SkillTimers["healing_potion"] != 0 {
						t.Fatal("rollback left partial potion effects")
					}
					return
				}
				if count != 1 || saved.Player.HP != 80 || saved.Revision != 1 || saved.Stats.PotionsUsed != 1 || saved.SkillTimers["healing_potion"] != 8 || saved.LastRequestID != request.RequestID {
					t.Fatal("potion consumption and healing diverged")
				}
				var writers int
				if err := database.QueryRowContext(ctx, `SELECT COUNT(DISTINCT xid) FROM (SELECT xmin::text xid FROM user_consumables WHERE client_uid=$1 UNION ALL SELECT xmin::text FROM app_meta WHERE key=$2) changes`, uid, key).Scan(&writers); err != nil {
					t.Fatal(err)
				}
				if writers != 1 {
					t.Fatal("potion inventory and snapshot did not commit together")
				}
			}
			assertState(committed)
			// Both lost COMMIT and lost ROLLBACK converge when the same request is retried.
			for i := 0; i < 2; i++ {
				if _, err = bot.updateRift(ctx, uid, request, run.Build, now.Add(time.Second)); err != nil {
					t.Fatal(err)
				}
				assertState(true)
			}
		})
	}
}
