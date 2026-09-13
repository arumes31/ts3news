//go:build integration

package db

import (
	"context"
	"database/sql"
	"testing"
)

func economyLedgerTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	database := goldResetTestDatabase(t)
	_, err := database.Exec(`ALTER TABLE users ADD COLUMN abyss_boss_tokens BIGINT NOT NULL DEFAULT 0;
		ALTER TABLE users ADD COLUMN scrap_stack INTEGER NOT NULL DEFAULT 0;
		CREATE TABLE user_materials (client_uid TEXT, mat_id TEXT, count BIGINT);
		CREATE TABLE user_consumables (client_uid TEXT, cons_id TEXT, remaining_fights INTEGER);
		INSERT INTO app_meta VALUES ('gold_economy_version','2'), ('economy_deployed_revision','test-revision');`)
	if err != nil {
		t.Fatal(err)
	}
	migration, err := migrationsFS.ReadFile("migrations/0105_economy_ledger.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}
	return database
}

func TestEconomyLedgerAtomicBalancesAndStock(t *testing.T) {
	database := economyLedgerTestDatabase(t)
	tx, err := database.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := SetEconomyContext(context.Background(), tx, "test.settlement", "req-1", "run-1", ""); err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(`UPDATE users SET gold=gold-100,abyss_tokens=abyss_tokens+1,abyss_boss_tokens=2,scrap_stack=3 WHERE client_uid='online';
		INSERT INTO user_materials VALUES ('online','dust',20);
		UPDATE user_materials SET count=7;
		DELETE FROM user_materials;
		INSERT INTO user_consumables VALUES ('online','potion',4);
		UPDATE user_consumables SET remaining_fights=remaining_fights;
		UPDATE users SET xp=xp+1 WHERE client_uid='online';`)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	checks := map[string]string{
		`SELECT COUNT(*)::text FROM economy_ledger`:                       "8",
		`SELECT COUNT(DISTINCT transaction_id)::text FROM economy_ledger`: "1",
		`SELECT COUNT(*)::text FROM economy_ledger WHERE source='test.settlement' AND request_id='req-1' AND run_id='run-1' AND economy_epoch='2' AND revision='test-revision'`: "8",
		`SELECT SUM(delta)::text FROM economy_ledger WHERE currency='material:dust'`:                                                                                            "0",
		`SELECT delta::text FROM economy_ledger WHERE currency='gold'`:                                                                                                          "-100",
		`SELECT balance_before::text FROM economy_ledger WHERE currency='gold'`:                                                                                                 "42438666400000000",
		`SELECT item_id FROM economy_ledger WHERE currency='consumable:potion'`:                                                                                                 "potion",
	}
	for query, want := range checks {
		if got := goldResetScalar(t, database, query); got != want {
			t.Errorf("%s: got %s, want %s", query, got, want)
		}
	}
}

func TestEconomyLedgerRollbackAndConnectionReuse(t *testing.T) {
	database := economyLedgerTestDatabase(t)
	database.SetMaxOpenConns(1)
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := SetEconomyContext(context.Background(), tx, "rolled.back", "private-request", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec("UPDATE users SET gold=gold+9 WHERE client_uid='online'"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if got := goldResetScalar(t, database, "SELECT COUNT(*)::text FROM economy_ledger"); got != "0" {
		t.Fatal(got)
	}
	if _, err := database.Exec("UPDATE users SET gold=gold+1 WHERE client_uid='online'"); err != nil {
		t.Fatal(err)
	}
	if got := goldResetScalar(t, database, "SELECT source || ':' || request_id FROM economy_ledger"); got != "unattributed:" {
		t.Fatal(got)
	}
}

func TestEconomyLedgerFailureRollsBackBalance(t *testing.T) {
	database := economyLedgerTestDatabase(t)
	if _, err := database.Exec("ALTER TABLE economy_ledger ADD CHECK (delta < 0)"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("UPDATE users SET gold=gold+1 WHERE client_uid='offline'"); err == nil {
		t.Fatal("balance write succeeded without its ledger entry")
	}
	if got := goldResetScalar(t, database, "SELECT gold::text FROM users WHERE client_uid='offline'"); got != "4500" {
		t.Fatal(got)
	}
}

func TestEconomyLedgerStockOwnershipTransferAndTalentCredit(t *testing.T) {
	database := economyLedgerTestDatabase(t)
	if _, err := database.Exec(`INSERT INTO user_materials VALUES ('online','dust',7);
		UPDATE user_materials SET client_uid='offline',mat_id='core';
		UPDATE users SET abyss_talent_credit=123 WHERE client_uid='offline';`); err != nil {
		t.Fatal(err)
	}
	for query, want := range map[string]string{
		`SELECT SUM(delta)::text FROM economy_ledger WHERE client_uid='online' AND currency='material:dust'`:  "0",
		`SELECT SUM(delta)::text FROM economy_ledger WHERE client_uid='offline' AND currency='material:core'`: "7",
		`SELECT delta::text FROM economy_ledger WHERE currency='abyss_talent_credit'`:                         "123",
	} {
		if got := goldResetScalar(t, database, query); got != want {
			t.Errorf("%s: got %s, want %s", query, got, want)
		}
	}
}

func TestEconomyLedgerStaticSourceTagDoesNotStoreSQL(t *testing.T) {
	database := economyLedgerTestDatabase(t)
	if _, err := database.Exec(`/* economy:bot.Bot.legacyReward */ UPDATE users SET gold=gold+3
		WHERE client_uid='online' AND 'do-not-log-secret' <> ''`); err != nil {
		t.Fatal(err)
	}
	if got := goldResetScalar(t, database, `SELECT source FROM economy_ledger`); got != "bot.Bot.legacyReward" {
		t.Fatal(got)
	}
	if got := goldResetScalar(t, database, `SELECT COUNT(*)::text FROM economy_ledger e WHERE to_jsonb(e)::text LIKE '%do-not-log-secret%' OR to_jsonb(e)::text LIKE '%UPDATE users%'`); got != "0" {
		t.Fatal("SQL or data appeared in ledger")
	}
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := SetEconomyContext(context.Background(), tx, "explicit.operation", "request", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`/* economy:bot.Bot.legacyReward */ UPDATE users SET gold=gold+4 WHERE client_uid='online'`); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if got := goldResetScalar(t, database, `SELECT source FROM economy_ledger ORDER BY id DESC LIMIT 1`); got != "explicit.operation" {
		t.Fatal(got)
	}
}

func TestEconomyLedgerDoesNotInterpretSQLLiteralAsSource(t *testing.T) {
	database := economyLedgerTestDatabase(t)
	if _, err := database.Exec(`UPDATE users SET gold=gold+1 WHERE client_uid='online'
		AND '/* economy:private.literal */' <> ''`); err != nil {
		t.Fatal(err)
	}
	if got := goldResetScalar(t, database, `SELECT source FROM economy_ledger`); got != "unattributed" {
		t.Fatalf("literal interpreted as source: %s", got)
	}
	if _, err := database.Exec(" \n\t/* economy:bot.Bot.actualSource */ UPDATE users SET gold=gold+1 WHERE client_uid='online'"); err != nil {
		t.Fatal(err)
	}
	if got := goldResetScalar(t, database, `SELECT source FROM economy_ledger ORDER BY id DESC LIMIT 1`); got != "bot.Bot.actualSource" {
		t.Fatal(got)
	}
}
