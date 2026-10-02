//go:build integration

package bot

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lib/pq"
	"ts3news/internal/content"
	appdb "ts3news/internal/db"
	"ts3news/internal/rift"
)

// Each test suite owns a new database; migrations never touch the supplied admin database.
func riftBankIntegrationDatabase(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dsn := os.Getenv("RIFT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set RIFT_TEST_DATABASE_URL to disposable PostgreSQL with CREATE DATABASE permission")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		t.Fatal("expected a PostgreSQL test URL")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("rift_bank_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec("CREATE DATABASE " + pq.QuoteIdentifier(name) + " TEMPLATE template0"); err != nil {
		_ = admin.Close()
		t.Fatal(err)
	}
	var database *sql.DB
	t.Cleanup(func() {
		if database != nil {
			_ = database.Close()
		}
		if _, err := admin.Exec("DROP DATABASE " + pq.QuoteIdentifier(name) + " WITH (FORCE)"); err != nil {
			t.Error(err)
		}
		_ = admin.Close()
	})
	parsed.Path = "/" + name
	query := parsed.Query()
	query.Del("search_path")
	parsed.RawQuery = query.Encode()
	database, err = sql.Open("postgres", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	if err = appdb.Migrate(database); err != nil {
		t.Fatal(err)
	}
	if _, err = database.Exec(`INSERT INTO app_meta(key,value) VALUES ('gold_economy_version','2'),('economy_deployed_revision','rift-commit-test') ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`); err != nil {
		t.Fatal(err)
	}
	return database, parsed.String()
}

// The first Commit loses its confirmation after either real COMMIT or real ROLLBACK.
// Closing the connection and returning ErrBadConn makes the application outcome unknown.
// SQL execution, migrations, triggers, inventory and snapshots are all real PostgreSQL.
type riftLostCommitConnector struct {
	driver.Connector
	committed bool
	fired     atomic.Bool
}
type riftLostCommitConn struct {
	driver.Conn
	control *riftLostCommitConnector
}
type riftLostCommitTx struct {
	driver.Tx
	conn    driver.Conn
	control *riftLostCommitConnector
}

func (c *riftLostCommitConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &riftLostCommitConn{Conn: conn, control: c}, nil
}
func (c *riftLostCommitConn) Begin() (driver.Tx, error) {
	tx, err := c.Conn.Begin()
	if err != nil {
		return nil, err
	}
	return &riftLostCommitTx{Tx: tx, conn: c.Conn, control: c.control}, nil
}
func (tx *riftLostCommitTx) Commit() error {
	if !tx.control.fired.CompareAndSwap(false, true) {
		return tx.Tx.Commit()
	}
	var err error
	if tx.control.committed {
		err = tx.Tx.Commit()
	} else {
		err = tx.Tx.Rollback()
	}
	if err != nil {
		return err
	}
	_ = tx.conn.Close()
	return driver.ErrBadConn
}

func TestRiftActualBankCommitUncertainty(t *testing.T) {
	database, dsn := riftBankIntegrationDatabase(t)
	for _, kind := range []string{"bank", "exit", "next", "advance"} {
		for _, committed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/committed=%t", kind, committed), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				uid := fmt.Sprintf("%s-%t", kind, committed)
				if _, err := database.ExecContext(ctx, "INSERT INTO users(client_uid,nickname,gold) VALUES($1,'Commit test',100)", uid); err != nil {
					t.Fatal(err)
				}
				now := time.Now()
				run := rift.NewRunAtLevel("run-"+uid, rift.Build{Name: "Commit test", HP: 200}, now, riftMobCatalog(now), 1)
				run.Room = 2
				run.Status = "cleared"
				run.Epoch = "2"
				run.Revision = 4
				run.Gold = 30
				gear := content.Gear{ID: "ABYSS_COMMIT_PROBE", Name: "Commit probe blade", Rarity: content.RarityRare, MaxDurability: 80, FoundBoss: "Rift Brawl commit test"}
				run.Drops = []rift.Drop{{ID: "fight-drop", Gold: 30, Collected: true, Gear: &gear}}
				run.Objectives = &rift.MissionObjectives{Mission: 1, Finished: true, RewardPerObjective: 5, Entries: []rift.ObjectiveProgress{{ID: "timed", Status: "complete"}, {ID: "guard", Status: "complete"}, {ID: "no_damage", Status: "failed"}}}
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
				request := riftRequest{Kind: kind, RunID: run.ID, Revision: 5, RequestID: "bank-request-" + uid}
				out, err := bot.updateRift(ctx, uid, request, rift.Build{}, now.Add(time.Second))
				if out != nil || !errors.Is(err, driver.ErrBadConn) || !fault.fired.Load() {
					t.Fatalf("expected unknown commit outcome, got %v %+v", err, out)
				}
				assertState := func(paid bool) *rift.Run {
					t.Helper()
					var gold, items, ledgerRows, delta, transactions, attributed int64
					if err := database.QueryRowContext(ctx, "SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold); err != nil {
						t.Fatal(err)
					}
					if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_inventory WHERE client_uid=$1", uid).Scan(&items); err != nil {
						t.Fatal(err)
					}
					if err := database.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(delta),0),COUNT(DISTINCT transaction_id),COUNT(*) FILTER(WHERE request_id=$2 AND run_id=$3 AND economy_epoch='2') FROM economy_ledger WHERE client_uid=$1 AND source='rift_brawl' AND currency='gold'`, uid, request.RequestID, run.ID).Scan(&ledgerRows, &delta, &transactions, &attributed); err != nil {
						t.Fatal(err)
					}
					saved, err := loadRift(ctx, database, uid)
					if err != nil {
						t.Fatal(err)
					}
					if !paid {
						if gold != 100 || items != 0 || ledgerRows != 0 || delta != 0 || saved.Revision != 4 || saved.Status != "cleared" || saved.Drops[0].Banked || saved.BankedGold != 0 {
							t.Fatalf("rollback left partial rewards: gold=%d items=%d ledger=%d run=%+v", gold, items, ledgerRows, saved)
						}
						return saved
					}
					if gold != 140 || items != 1 || ledgerRows != 2 || delta != 40 || transactions != 1 || attributed != 2 {
						t.Fatalf("non-atomic or duplicate settlement: gold=%d items=%d ledger=%d delta=%d transactions=%d attributed=%d", gold, items, ledgerRows, delta, transactions, attributed)
					}
					if saved.Revision != 5 || saved.LastRequestID != request.RequestID || saved.BankedGold != 40 || saved.BankedObjectiveGold != 10 || saved.TotalBankedItems() != 1 || saved.Gold != 0 {
						t.Fatalf("snapshot differs from delivery: %+v", saved)
					}
					var itemData string
					var durability int
					if err := database.QueryRowContext(ctx, "SELECT item_data,durability FROM user_inventory WHERE client_uid=$1", uid).Scan(&itemData, &durability); err != nil {
						t.Fatal(err)
					}
					var delivered content.Gear
					if err := json.Unmarshal([]byte(itemData), &delivered); err != nil {
						t.Fatal(err)
					}
					if delivered.ID != gear.ID || delivered.Name != gear.Name || delivered.FoundBoss != gear.FoundBoss || durability != 80 {
						t.Fatal("delivered gear lost identity/provenance")
					}
					var writers int
					if err := database.QueryRowContext(ctx, `SELECT COUNT(DISTINCT xid) FROM (SELECT xmin::text xid FROM users WHERE client_uid=$1 UNION ALL SELECT xmin::text FROM user_inventory WHERE client_uid=$1 UNION ALL SELECT xmin::text FROM app_meta WHERE key=$2) changes`, uid, key).Scan(&writers); err != nil {
						t.Fatal(err)
					}
					if writers != 1 {
						t.Fatal("balance, item and snapshot did not commit together")
					}
					return saved
				}
				assertState(committed)
				// Explicit recovery reuses the exact request. Both possible outcomes must converge.
				confirmed, err := bot.updateRift(ctx, uid, request, rift.Build{}, now.Add(2*time.Second))
				if err != nil || confirmed == nil {
					t.Fatalf("recovery failed: %v", err)
				}
				settled := assertState(true)
				before, err := encodeRift(settled)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = bot.updateRift(ctx, uid, request, rift.Build{}, now.Add(3*time.Second)); err != nil {
					t.Fatal(err)
				}
				after, err := encodeRift(assertState(true))
				if err != nil {
					t.Fatal(err)
				}
				if before != after {
					t.Fatal("late replay altered the confirmed snapshot")
				}
			})
		}
	}
}
