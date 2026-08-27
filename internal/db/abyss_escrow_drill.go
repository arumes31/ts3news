package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// AbyssEscrowIntegrity reports authoritative row totals and violations that
// database constraints alone cannot express across the escrow tables.
type AbyssEscrowIntegrity struct {
	Counts                 AbyssEscrowSnapshotCounts `json:"counts"`
	OrphanLoot             int                       `json:"orphan_loot"`
	OrphanSessions         int                       `json:"orphan_sessions"`
	SessionsWithoutOwner   int                       `json:"sessions_without_owner_member"`
	NonObjectLootPayloads  int                       `json:"non_object_loot_payloads"`
	MembersWithoutSessions int                       `json:"members_without_sessions"`
}

func (report AbyssEscrowIntegrity) Healthy() bool {
	return report.OrphanLoot == 0 &&
		report.OrphanSessions == 0 &&
		report.SessionsWithoutOwner == 0 &&
		report.NonObjectLootPayloads == 0 &&
		report.MembersWithoutSessions == 0
}

// CheckAbyssEscrowIntegrity performs one read-only query so every reported
// count describes the same database statement snapshot.
func CheckAbyssEscrowIntegrity(ctx context.Context, database *sql.DB) (AbyssEscrowIntegrity, error) {
	if database == nil {
		return AbyssEscrowIntegrity{}, errors.New("checking Abyss escrow integrity: nil database")
	}
	const query = `SELECT
  (SELECT COUNT(*) FROM abyss_active),
  (SELECT COUNT(*) FROM abyss_escrow_loot),
  (SELECT COUNT(*) FROM abyss_combat_sessions),
  (SELECT COUNT(*) FROM abyss_combat_members),
  (SELECT COUNT(*) FROM abyss_escrow_loot l WHERE NOT EXISTS (SELECT 1 FROM abyss_active a WHERE a.client_uid = l.client_uid)),
  (SELECT COUNT(*) FROM abyss_combat_sessions s WHERE NOT EXISTS (SELECT 1 FROM abyss_active a WHERE a.client_uid = s.owner_uid)),
  (SELECT COUNT(*) FROM abyss_combat_sessions s WHERE NOT EXISTS (SELECT 1 FROM abyss_combat_members m WHERE m.session_id = s.session_id AND m.client_uid = s.owner_uid)),
  (SELECT COUNT(*) FROM abyss_escrow_loot WHERE jsonb_typeof(item_data) IS DISTINCT FROM 'object'),
  (SELECT COUNT(*) FROM abyss_combat_members m WHERE NOT EXISTS (SELECT 1 FROM abyss_combat_sessions s WHERE s.session_id = m.session_id))`
	var report AbyssEscrowIntegrity
	err := database.QueryRowContext(ctx, query).Scan(
		&report.Counts.Active,
		&report.Counts.Loot,
		&report.Counts.Sessions,
		&report.Counts.Members,
		&report.OrphanLoot,
		&report.OrphanSessions,
		&report.SessionsWithoutOwner,
		&report.NonObjectLootPayloads,
		&report.MembersWithoutSessions,
	)
	if err != nil {
		return AbyssEscrowIntegrity{}, fmt.Errorf("querying Abyss escrow integrity: %w", err)
	}
	return report, nil
}

// DrillAbyssEscrowRestore loads a verified snapshot into temporary tables that
// inherit the current production schema, compares the round-tripped payload,
// then rolls the entire transaction back. It never writes the live tables.
func DrillAbyssEscrowRestore(ctx context.Context, database *sql.DB, snapshot AbyssEscrowSnapshot) (AbyssEscrowSnapshotCounts, error) {
	if database == nil {
		return AbyssEscrowSnapshotCounts{}, errors.New("drilling Abyss escrow restore: nil database")
	}
	if err := ValidateAbyssEscrowSnapshot(snapshot); err != nil {
		return AbyssEscrowSnapshotCounts{}, fmt.Errorf("validating Abyss escrow restore drill input: %w", err)
	}
	if err := normalizeAbyssSnapshotTables(&snapshot); err != nil {
		return AbyssEscrowSnapshotCounts{}, err
	}

	tx, err := database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return AbyssEscrowSnapshotCounts{}, fmt.Errorf("starting Abyss escrow restore drill: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	createStatements := []string{
		`CREATE TEMP TABLE abyss_active_restore_drill (LIKE abyss_active INCLUDING ALL) ON COMMIT DROP`,
		`CREATE TEMP TABLE abyss_escrow_loot_restore_drill (LIKE abyss_escrow_loot INCLUDING ALL) ON COMMIT DROP`,
		`CREATE TEMP TABLE abyss_combat_sessions_restore_drill (LIKE abyss_combat_sessions INCLUDING ALL) ON COMMIT DROP`,
		`CREATE TEMP TABLE abyss_combat_members_restore_drill (LIKE abyss_combat_members INCLUDING ALL) ON COMMIT DROP`,
	}
	for _, statement := range createStatements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return AbyssEscrowSnapshotCounts{}, fmt.Errorf("creating Abyss escrow restore drill table: %w", err)
		}
	}

	restoreStatements := []struct {
		name    string
		query   string
		payload json.RawMessage
		want    int
	}{
		{name: "abyss_active", query: `INSERT INTO abyss_active_restore_drill SELECT * FROM jsonb_populate_recordset(NULL::abyss_active, $1::jsonb)`, payload: snapshot.Active, want: snapshot.Counts.Active},
		{name: "abyss_escrow_loot", query: `INSERT INTO abyss_escrow_loot_restore_drill SELECT * FROM jsonb_populate_recordset(NULL::abyss_escrow_loot, $1::jsonb)`, payload: snapshot.Loot, want: snapshot.Counts.Loot},
		{name: "abyss_combat_sessions", query: `INSERT INTO abyss_combat_sessions_restore_drill SELECT * FROM jsonb_populate_recordset(NULL::abyss_combat_sessions, $1::jsonb)`, payload: snapshot.Sessions, want: snapshot.Counts.Sessions},
		{name: "abyss_combat_members", query: `INSERT INTO abyss_combat_members_restore_drill SELECT * FROM jsonb_populate_recordset(NULL::abyss_combat_members, $1::jsonb)`, payload: snapshot.Members, want: snapshot.Counts.Members},
	}
	for _, item := range restoreStatements {
		result, err := tx.ExecContext(ctx, item.query, []byte(item.payload))
		if err != nil {
			return AbyssEscrowSnapshotCounts{}, fmt.Errorf("restoring %s into drill table: %w", item.name, err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return AbyssEscrowSnapshotCounts{}, fmt.Errorf("reading %s drill row count: %w", item.name, err)
		}
		if rows != int64(item.want) {
			return AbyssEscrowSnapshotCounts{}, fmt.Errorf("restoring %s into drill table inserted %d rows, want %d", item.name, rows, item.want)
		}
	}

	roundTrip := AbyssEscrowSnapshot{Version: snapshot.Version, CreatedAt: snapshot.CreatedAt, Counts: snapshot.Counts}
	queries := []struct {
		name   string
		query  string
		target *json.RawMessage
	}{
		{name: "abyss_active", query: `SELECT COALESCE(jsonb_agg(to_jsonb(row_data) ORDER BY row_data.client_uid), '[]'::jsonb) FROM abyss_active_restore_drill AS row_data`, target: &roundTrip.Active},
		{name: "abyss_escrow_loot", query: `SELECT COALESCE(jsonb_agg(to_jsonb(row_data) ORDER BY row_data.id), '[]'::jsonb) FROM abyss_escrow_loot_restore_drill AS row_data`, target: &roundTrip.Loot},
		{name: "abyss_combat_sessions", query: `SELECT COALESCE(jsonb_agg(to_jsonb(row_data) ORDER BY row_data.session_id), '[]'::jsonb) FROM abyss_combat_sessions_restore_drill AS row_data`, target: &roundTrip.Sessions},
		{name: "abyss_combat_members", query: `SELECT COALESCE(jsonb_agg(to_jsonb(row_data) ORDER BY row_data.session_id, row_data.client_uid), '[]'::jsonb) FROM abyss_combat_members_restore_drill AS row_data`, target: &roundTrip.Members},
	}
	for _, item := range queries {
		var encoded []byte
		if err := tx.QueryRowContext(ctx, item.query).Scan(&encoded); err != nil {
			return AbyssEscrowSnapshotCounts{}, fmt.Errorf("reading %s restore drill rows: %w", item.name, err)
		}
		*item.target = append((*item.target)[:0], encoded...)
	}
	if err := normalizeAbyssSnapshotTables(&roundTrip); err != nil {
		return AbyssEscrowSnapshotCounts{}, fmt.Errorf("normalizing Abyss escrow restore drill result: %w", err)
	}
	if abyssEscrowSnapshotChecksum(roundTrip) != snapshot.Checksum {
		return AbyssEscrowSnapshotCounts{}, errors.New("abyss escrow restore drill round-trip checksum mismatch")
	}
	if err := tx.Rollback(); err != nil {
		return AbyssEscrowSnapshotCounts{}, fmt.Errorf("rolling back Abyss escrow restore drill: %w", err)
	}
	return snapshot.Counts, nil
}
