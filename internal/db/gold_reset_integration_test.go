//go:build integration

package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lib/pq"
)

func goldResetTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("GOLD_RESET_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set GOLD_RESET_TEST_DATABASE_URL to a disposable PostgreSQL database")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("gold_reset_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + pq.QuoteIdentifier(schema)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP SCHEMA " + pq.QuoteIdentifier(schema) + " CASCADE"); err != nil {
			t.Error(err)
		}
		if err := admin.Close(); err != nil {
			t.Error(err)
		}
	})
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	database, err := sql.Open("postgres", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Error(err)
		}
	})
	_, err = database.Exec(`
		CREATE TABLE users (client_uid TEXT PRIMARY KEY, gold BIGINT NOT NULL, xp BIGINT, level INT, abyss_tokens BIGINT);
		CREATE TABLE abyss_active (client_uid TEXT PRIMARY KEY, escrow BIGINT NOT NULL, depth INT, insured INT, event_state JSONB);
		CREATE TABLE abyss_escrow_loot (id BIGINT PRIMARY KEY, client_uid TEXT, item_type TEXT, item_data JSONB);
		CREATE TABLE app_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		CREATE TABLE user_gear (client_uid TEXT, item_id TEXT);
		CREATE TABLE abyss_combat_sessions (session_id TEXT, state JSONB);
		INSERT INTO users VALUES ('online',42438666400000000,987654,100,88),('offline',4500,1200,10,12);
		INSERT INTO abyss_active VALUES ('online',9000000000000001,400,50,
			'{"type":"echo_floor","previous_reward":9007199254740993,"echo_reward":99,"purchased":true}');
		INSERT INTO abyss_escrow_loot VALUES (1,'online','gold','{"type":"gold","gold":123}'),
			(2,'online','gear','{"id":"sword"}'),(3,'online','tokens','{"tokens":10}');
		INSERT INTO user_gear VALUES ('online','paid-sword');
		INSERT INTO abyss_combat_sessions VALUES ('saved-fight','{"result":{"gold":9000}}');
		INSERT INTO app_meta VALUES ('abyss_echo_seed_online','9007199254740993'),
			('abyss_run_flags_online','{"echo_original_reward":9007199254740993,"run_bounty_reward":9007199254740993,"vault_keys":3}'),
			('abyss_deferred_event_online','{"state":"{\"type\":\"echo_floor\",\"previous_reward\":123,\"echo_reward\":45,\"purchased\":true}","origin_depth":390,"expires_depth":410}'),
			('abyss_deferred_event_offline','{"state":"{\"type\":\"market\",\"gold\":123,\"purchased\":true}","origin_depth":2}'),
			('abyssXecho_seed_online','777'),('purchase_receipt_online','keep-me');`)
	if err != nil {
		t.Fatal(err)
	}
	return database
}

func goldResetScalar(t *testing.T, database *sql.DB, query string) string {
	t.Helper()
	var value string
	if err := database.QueryRow(query).Scan(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestGoldResetIntegrationVersionAndPreservation(t *testing.T) {
	database := goldResetTestDatabase(t)
	ctx := context.Background()
	applied, err := applyGoldEconomyVersion(ctx, database, 1)
	if err != nil || !applied {
		t.Fatalf("first reset = %v, %v", applied, err)
	}
	checks := map[string]string{
		`SELECT SUM(gold)::text FROM users`:                                                                            "0",
		`SELECT escrow::text FROM abyss_active`:                                                                        "0",
		`SELECT depth || '/' || insured FROM abyss_active`:                                                             "400/50",
		`SELECT xp || '/' || level || '/' || abyss_tokens FROM users WHERE client_uid='online'`:                        "987654/100/88",
		`SELECT string_agg(item_type,',' ORDER BY id) FROM abyss_escrow_loot`:                                          "gear,tokens",
		`SELECT item_id FROM user_gear`:                                                                                "paid-sword",
		`SELECT COUNT(*)::text FROM abyss_combat_sessions`:                                                             "1",
		`SELECT COUNT(*)::text FROM app_meta WHERE key='abyss_echo_seed_online'`:                                       "0",
		`SELECT value FROM app_meta WHERE key='abyssXecho_seed_online'`:                                                "777",
		`SELECT value FROM app_meta WHERE key='purchase_receipt_online'`:                                               "keep-me",
		`SELECT value::jsonb->>'echo_original_reward' FROM app_meta WHERE key='abyss_run_flags_online'`:                "0",
		`SELECT value::jsonb->>'run_bounty_reward' FROM app_meta WHERE key='abyss_run_flags_online'`:                   "9007199254740993",
		`SELECT event_state->>'echo_reward' FROM abyss_active`:                                                         "0",
		`SELECT event_state->>'purchased' FROM abyss_active`:                                                           "true",
		`SELECT (value::jsonb->>'state')::jsonb->>'echo_reward' FROM app_meta WHERE key='abyss_deferred_event_online'`: "0",
		`SELECT value::jsonb->>'expires_depth' FROM app_meta WHERE key='abyss_deferred_event_online'`:                  "410",
		`SELECT (value::jsonb->>'state')::jsonb->>'gold' FROM app_meta WHERE key='abyss_deferred_event_offline'`:       "123",
		`SELECT value::jsonb->>'gold' FROM app_meta WHERE key='gold_reset_backup_1:user:online'`:                       "42438666400000000",
		`SELECT value::jsonb->>'escrow' FROM app_meta WHERE key='gold_reset_backup_1:run:online'`:                      "9000000000000001",
		`SELECT value::jsonb->'item_data'->>'gold' FROM app_meta WHERE key='gold_reset_backup_1:loot:1'`:               "123",
		`SELECT value FROM app_meta WHERE key='gold_reset_backup_1:meta:abyss_echo_seed_online'`:                       "9007199254740993",
	}
	for query, want := range checks {
		if got := goldResetScalar(t, database, query); got != want {
			t.Errorf("%s = %q, want %q", query, got, want)
		}
	}
	if _, err := database.Exec(`UPDATE users SET gold=123; UPDATE abyss_active SET escrow=456;
		INSERT INTO users VALUES ('new-player',789,0,1,0)`); err != nil {
		t.Fatal(err)
	}
	if applied, err := applyGoldEconomyVersion(ctx, database, 1); err != nil || applied {
		t.Fatalf("repeat = %v, %v", applied, err)
	}
	if got := goldResetScalar(t, database, `SELECT SUM(gold)::text FROM users`); got != "1035" {
		t.Fatalf("repeat wiped new gold: %s", got)
	}
	if got := goldResetScalar(t, database, `SELECT escrow::text FROM abyss_active`); got != "456" {
		t.Fatalf("repeat wiped new cache: %s", got)
	}
	if applied, err := applyGoldEconomyVersion(ctx, database, 2); err != nil || !applied {
		t.Fatalf("upgrade = %v, %v", applied, err)
	}
	if got := goldResetScalar(t, database, `SELECT value::jsonb->>'gold' FROM app_meta WHERE key='gold_reset_backup_2:user:new-player'`); got != "789" {
		t.Fatal(got)
	}
	if _, err := database.Exec(`UPDATE users SET gold=99`); err != nil {
		t.Fatal(err)
	}
	if applied, err := applyGoldEconomyVersion(ctx, database, 1); err != nil || applied {
		t.Fatalf("downgrade = %v, %v", applied, err)
	}
	if got := goldResetScalar(t, database, `SELECT gold::text FROM users WHERE client_uid='online'`); got != "99" {
		t.Fatal(got)
	}
}

func TestGoldResetIntegrationRollbackAndRetry(t *testing.T) {
	database := goldResetTestDatabase(t)
	if _, err := database.Exec(`ALTER TABLE abyss_active ADD CONSTRAINT refuse_reset CHECK (escrow>0)`); err != nil {
		t.Fatal(err)
	}
	if applied, err := applyGoldEconomyVersion(context.Background(), database, 1); err == nil || applied {
		t.Fatalf("failed write = %v, %v", applied, err)
	}
	if got := goldResetScalar(t, database, `SELECT gold::text FROM users WHERE client_uid='online'`); got != "42438666400000000" {
		t.Fatal("wallet was not rolled back: " + got)
	}
	if got := goldResetScalar(t, database, `SELECT COUNT(*)::text FROM app_meta WHERE key LIKE 'gold_%'`); got != "0" {
		t.Fatal("failed reset left a version/backup: " + got)
	}
	if _, err := database.Exec(`ALTER TABLE abyss_active DROP CONSTRAINT refuse_reset`); err != nil {
		t.Fatal(err)
	}
	if applied, err := applyGoldEconomyVersion(context.Background(), database, 1); err != nil || !applied {
		t.Fatalf("retry = %v, %v", applied, err)
	}
}

func TestGoldResetIntegrationConcurrentStartup(t *testing.T) {
	database := goldResetTestDatabase(t)
	var wait sync.WaitGroup
	results := make(chan bool, 4)
	errors := make(chan error, 4)
	for range 4 {
		wait.Go(func() {
			applied, err := EnsureGoldEconomyVersion(context.Background(), database)
			results <- applied
			errors <- err
		})
	}
	wait.Wait()
	close(results)
	close(errors)
	count := 0
	for applied := range results {
		if applied {
			count++
		}
	}
	for err := range errors {
		if err != nil {
			t.Error(err)
		}
	}
	if count != 1 {
		t.Fatalf("reset applied %d times", count)
	}
}

func TestGoldResetIntegrationInvalidStoredVersion(t *testing.T) {
	database := goldResetTestDatabase(t)
	if _, err := database.Exec(`INSERT INTO app_meta VALUES ('gold_economy_version','broken')`); err != nil {
		t.Fatal(err)
	}
	if applied, err := EnsureGoldEconomyVersion(context.Background(), database); err == nil || applied || !strings.Contains(err.Error(), "version") {
		t.Fatalf("invalid marker = %v, %v", applied, err)
	}
	if got := goldResetScalar(t, database, `SELECT gold::text FROM users WHERE client_uid='online'`); got != "42438666400000000" {
		t.Fatal(got)
	}
}

func TestGoldResetIntegrationMalformedEchoRollsBack(t *testing.T) {
	database := goldResetTestDatabase(t)
	if _, err := database.Exec(`UPDATE app_meta SET value='not-json' WHERE key='abyss_run_flags_online'`); err != nil {
		t.Fatal(err)
	}
	if applied, err := EnsureGoldEconomyVersion(context.Background(), database); err == nil || applied {
		t.Fatalf("malformed echo = %v, %v", applied, err)
	}
	checks := map[string]string{
		`SELECT gold::text FROM users WHERE client_uid='online'`:              "42438666400000000",
		`SELECT escrow::text FROM abyss_active`:                               "9000000000000001",
		`SELECT COUNT(*)::text FROM abyss_escrow_loot WHERE item_type='gold'`: "1",
		`SELECT value FROM app_meta WHERE key='abyss_echo_seed_online'`:       "9007199254740993",
		`SELECT COUNT(*)::text FROM app_meta WHERE key LIKE 'gold_%'`:         "0",
	}
	for query, want := range checks {
		if got := goldResetScalar(t, database, query); got != want {
			t.Errorf("%s = %q, want %q", query, got, want)
		}
	}
}
