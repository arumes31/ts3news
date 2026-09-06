package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
)

// GoldEconomyVersion is monotonic. Increment it only for an intentional global
// wallet and Abyss gold reset, deployed with all previous bot instances stopped.
const GoldEconomyVersion = 1

// EnsureGoldEconomyVersion applies the current gold reset once to all accounts,
// including offline accounts. Call after migrations and before starting activity.
// Backups, currency changes and the version marker commit as one transaction.
func EnsureGoldEconomyVersion(ctx context.Context, database *sql.DB) (bool, error) {
	return applyGoldEconomyVersion(ctx, database, GoldEconomyVersion)
}

func applyGoldEconomyVersion(ctx context.Context, database *sql.DB, version int) (applied bool, err error) {
	if version < 1 {
		return false, fmt.Errorf("gold economy version must be positive: %d", version)
	}
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("beginning gold economy reset: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			err = errors.Join(err, fmt.Errorf("rolling back gold economy reset: %w", rollbackErr))
			applied = false
		}
	}()
	// Insert first: SELECT FOR UPDATE alone cannot lock a missing version row.
	if _, err := tx.ExecContext(ctx, `INSERT INTO app_meta (key,value)
		VALUES ('gold_economy_version','0') ON CONFLICT (key) DO NOTHING`); err != nil {
		return false, fmt.Errorf("initializing gold economy version: %w", err)
	}
	var stored string
	if err := tx.QueryRowContext(ctx, `SELECT value FROM app_meta
		WHERE key='gold_economy_version' FOR UPDATE`).Scan(&stored); err != nil {
		return false, fmt.Errorf("locking gold economy version: %w", err)
	}
	current, err := strconv.Atoi(stored)
	if err != nil || current < 0 {
		return false, fmt.Errorf("invalid stored gold economy version %q", stored)
	}
	if current >= version {
		return false, nil
	}
	// Stabilize currency rows while taking backups. The old application must be
	// stopped: no database lock can invalidate its in-memory combat snapshots.
	if _, err := tx.ExecContext(ctx, `LOCK TABLE users, abyss_active, abyss_escrow_loot
		IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return false, fmt.Errorf("locking gold reset balances: %w", err)
	}
	prefix := "gold_reset_backup_" + strconv.Itoa(version) + ":"
	steps := []struct {
		name  string
		query string
		args  []any
	}{
		{"backing up wallets", `INSERT INTO app_meta (key,value)
			SELECT $1 || client_uid, jsonb_build_object('gold',gold)::text FROM users`, []any{prefix + "user:"}},
		{"backing up active runs", `INSERT INTO app_meta (key,value)
			SELECT $1 || client_uid, to_jsonb(a)::text FROM abyss_active a`, []any{prefix + "run:"}},
		{"backing up pending gold drops", `INSERT INTO app_meta (key,value)
			SELECT $1 || id::text, to_jsonb(l)::text FROM abyss_escrow_loot l
			WHERE item_type='gold' OR item_data->>'type'='gold'`, []any{prefix + "loot:"}},
		{"backing up echo metadata", `INSERT INTO app_meta (key,value)
			SELECT $1 || key, value FROM app_meta
			WHERE LEFT(key,LENGTH('abyss_echo_seed_'))='abyss_echo_seed_'
			   OR LEFT(key,LENGTH('abyss_run_flags_'))='abyss_run_flags_'
			   OR LEFT(key,LENGTH('abyss_deferred_event_'))='abyss_deferred_event_'`, []any{prefix + "meta:"}},
		{"resetting wallets", `UPDATE users SET gold=0 WHERE gold<>0`, nil},
		{"resetting active caches", `UPDATE abyss_active SET escrow=0,
			event_state=CASE WHEN event_state->>'type'='echo_floor'
			THEN event_state || '{"previous_reward":0,"echo_reward":0}'::jsonb
			ELSE event_state END`, nil},
		{"resetting pending gold drops", `DELETE FROM abyss_escrow_loot
			WHERE item_type='gold' OR item_data->>'type'='gold'`, nil},
		{"resetting echo seeds", `DELETE FROM app_meta
			WHERE LEFT(key,LENGTH('abyss_echo_seed_'))='abyss_echo_seed_'`, nil},
		{"resetting repeated echo rewards", `UPDATE app_meta
			SET value=jsonb_set(value::jsonb,'{echo_original_reward}','0'::jsonb,false)::text
			WHERE LEFT(key,LENGTH('abyss_run_flags_'))='abyss_run_flags_'`, nil},
		// CASE ensures arbitrary metadata values are never cast to JSON. Deferred
		// event state is a JSON-encoded string, not an embedded JSON object.
		{"resetting deferred echo displays", `UPDATE app_meta SET value=jsonb_set(value::jsonb,'{state}',
			to_jsonb(((value::jsonb->>'state')::jsonb || '{"previous_reward":0,"echo_reward":0}'::jsonb)::text),false)::text
			WHERE CASE WHEN LEFT(key,LENGTH('abyss_deferred_event_'))='abyss_deferred_event_'
			THEN (value::jsonb->>'state')::jsonb->>'type'='echo_floor' ELSE false END`, nil},
		{"recording reset history", `INSERT INTO app_meta (key,value)
			VALUES ($1,jsonb_build_object('version',$2::integer,'applied_at',CURRENT_TIMESTAMP,
			'scope','wallet-and-abyss-gold')::text)`, []any{"gold_reset_applied_" + strconv.Itoa(version), version}},
		{"advancing gold economy version", `UPDATE app_meta SET value=$1
			WHERE key='gold_economy_version'`, []any{strconv.Itoa(version)}},
	}
	for _, step := range steps {
		if _, err := tx.ExecContext(ctx, step.query, step.args...); err != nil {
			return false, fmt.Errorf("%s: %w", step.name, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("committing gold economy reset: %w", err)
	}
	return true, nil
}
