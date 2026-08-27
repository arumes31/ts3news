package db

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const AbyssEscrowSnapshotVersion = 1

// AbyssEscrowSnapshot is a versioned, checksummed copy of every table needed
// to recover active Abyss escrow and its in-progress live combat ownership.
// Rows remain JSON objects so additive database columns are retained.
type AbyssEscrowSnapshot struct {
	Version   int                       `json:"version"`
	CreatedAt time.Time                 `json:"created_at"`
	Counts    AbyssEscrowSnapshotCounts `json:"counts"`
	Active    json.RawMessage           `json:"abyss_active"`
	Loot      json.RawMessage           `json:"abyss_escrow_loot"`
	Sessions  json.RawMessage           `json:"abyss_combat_sessions"`
	Members   json.RawMessage           `json:"abyss_combat_members"`
	Checksum  string                    `json:"checksum_sha256"`
}

type AbyssEscrowSnapshotCounts struct {
	Active   int `json:"abyss_active"`
	Loot     int `json:"abyss_escrow_loot"`
	Sessions int `json:"abyss_combat_sessions"`
	Members  int `json:"abyss_combat_members"`
}

type abyssSnapshotActiveKey struct {
	ClientUID string `json:"client_uid"`
}

type abyssSnapshotLootKey struct {
	ID        int64           `json:"id"`
	ClientUID string          `json:"client_uid"`
	ItemData  json.RawMessage `json:"item_data"`
}

type abyssSnapshotSessionKey struct {
	SessionID string `json:"session_id"`
	OwnerUID  string `json:"owner_uid"`
}

type abyssSnapshotMemberKey struct {
	SessionID string `json:"session_id"`
	ClientUID string `json:"client_uid"`
}

// ExportAbyssEscrowSnapshot reads a transactionally consistent snapshot. The
// returned value is normalized and checksummed. Call ValidateAbyssEscrowSnapshot
// separately so already-corrupt state can still be preserved for diagnosis.
func ExportAbyssEscrowSnapshot(ctx context.Context, database *sql.DB, now time.Time) (AbyssEscrowSnapshot, error) {
	if database == nil {
		return AbyssEscrowSnapshot{}, errors.New("exporting Abyss escrow snapshot: nil database")
	}
	tx, err := database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return AbyssEscrowSnapshot{}, fmt.Errorf("starting Abyss escrow snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	snapshot := AbyssEscrowSnapshot{Version: AbyssEscrowSnapshotVersion, CreatedAt: now.UTC()}
	queries := []struct {
		name   string
		query  string
		target *json.RawMessage
	}{
		{name: "abyss_active", query: `SELECT COALESCE(jsonb_agg(to_jsonb(row_data) ORDER BY row_data.client_uid), '[]'::jsonb) FROM abyss_active AS row_data`, target: &snapshot.Active},
		{name: "abyss_escrow_loot", query: `SELECT COALESCE(jsonb_agg(to_jsonb(row_data) ORDER BY row_data.id), '[]'::jsonb) FROM abyss_escrow_loot AS row_data`, target: &snapshot.Loot},
		{name: "abyss_combat_sessions", query: `SELECT COALESCE(jsonb_agg(to_jsonb(row_data) ORDER BY row_data.session_id), '[]'::jsonb) FROM abyss_combat_sessions AS row_data`, target: &snapshot.Sessions},
		{name: "abyss_combat_members", query: `SELECT COALESCE(jsonb_agg(to_jsonb(row_data) ORDER BY row_data.session_id, row_data.client_uid), '[]'::jsonb) FROM abyss_combat_members AS row_data`, target: &snapshot.Members},
	}
	for _, item := range queries {
		var encoded []byte
		if err := tx.QueryRowContext(ctx, item.query).Scan(&encoded); err != nil {
			return AbyssEscrowSnapshot{}, fmt.Errorf("reading %s for Abyss escrow snapshot: %w", item.name, err)
		}
		*item.target = append((*item.target)[:0], encoded...)
	}
	if err := sealAbyssEscrowSnapshot(&snapshot); err != nil {
		return AbyssEscrowSnapshot{}, err
	}
	if err := tx.Commit(); err != nil {
		return AbyssEscrowSnapshot{}, fmt.Errorf("committing Abyss escrow snapshot read: %w", err)
	}
	return snapshot, nil
}

// ValidateAbyssEscrowSnapshot verifies format, checksum, row counts, unique
// ownership keys, and cross-table references before any database drill begins.
func ValidateAbyssEscrowSnapshot(snapshot AbyssEscrowSnapshot) error {
	if err := validateAbyssEscrowSnapshotEnvelope(&snapshot); err != nil {
		return err
	}

	var active []abyssSnapshotActiveKey
	var loot []abyssSnapshotLootKey
	var sessions []abyssSnapshotSessionKey
	var members []abyssSnapshotMemberKey
	if err := decodeAbyssSnapshotTable("abyss_active", snapshot.Active, &active); err != nil {
		return err
	}
	if err := decodeAbyssSnapshotTable("abyss_escrow_loot", snapshot.Loot, &loot); err != nil {
		return err
	}
	if err := decodeAbyssSnapshotTable("abyss_combat_sessions", snapshot.Sessions, &sessions); err != nil {
		return err
	}
	if err := decodeAbyssSnapshotTable("abyss_combat_members", snapshot.Members, &members); err != nil {
		return err
	}

	activeUIDs := make(map[string]struct{}, len(active))
	for _, row := range active {
		if row.ClientUID == "" {
			return errors.New("abyss_active snapshot row has an empty client_uid")
		}
		if _, exists := activeUIDs[row.ClientUID]; exists {
			return fmt.Errorf("abyss_active snapshot contains duplicate client_uid %q", row.ClientUID)
		}
		activeUIDs[row.ClientUID] = struct{}{}
	}
	lootIDs := make(map[int64]struct{}, len(loot))
	for _, row := range loot {
		if row.ID <= 0 {
			return fmt.Errorf("abyss_escrow_loot snapshot contains invalid id %d", row.ID)
		}
		if _, exists := lootIDs[row.ID]; exists {
			return fmt.Errorf("abyss_escrow_loot snapshot contains duplicate id %d", row.ID)
		}
		lootIDs[row.ID] = struct{}{}
		if _, exists := activeUIDs[row.ClientUID]; !exists {
			return fmt.Errorf("abyss_escrow_loot snapshot id %d has no active run", row.ID)
		}
		if !jsonObject(row.ItemData) {
			return fmt.Errorf("abyss_escrow_loot snapshot id %d has non-object item_data", row.ID)
		}
	}
	sessionIDs := make(map[string]struct{}, len(sessions))
	for _, row := range sessions {
		if row.SessionID == "" || row.OwnerUID == "" {
			return errors.New("abyss_combat_sessions snapshot row has an empty identity")
		}
		if _, exists := sessionIDs[row.SessionID]; exists {
			return fmt.Errorf("abyss_combat_sessions snapshot contains duplicate session_id %q", row.SessionID)
		}
		sessionIDs[row.SessionID] = struct{}{}
		if _, exists := activeUIDs[row.OwnerUID]; !exists {
			return fmt.Errorf("abyss_combat_sessions snapshot %q has no active owner run", row.SessionID)
		}
	}
	memberKeys := make(map[string]struct{}, len(members))
	for _, row := range members {
		if row.SessionID == "" || row.ClientUID == "" {
			return errors.New("abyss_combat_members snapshot row has an empty identity")
		}
		if _, exists := sessionIDs[row.SessionID]; !exists {
			return fmt.Errorf("abyss_combat_members snapshot references missing session %q", row.SessionID)
		}
		key := row.SessionID + "\x00" + row.ClientUID
		if _, exists := memberKeys[key]; exists {
			return fmt.Errorf("abyss_combat_members snapshot contains duplicate member %q", row.ClientUID)
		}
		memberKeys[key] = struct{}{}
	}
	for _, row := range sessions {
		if _, exists := memberKeys[row.SessionID+"\x00"+row.OwnerUID]; !exists {
			return fmt.Errorf("abyss_combat_sessions snapshot %q has no owner member", row.SessionID)
		}
	}
	return nil
}

func validateAbyssEscrowSnapshotEnvelope(snapshot *AbyssEscrowSnapshot) error {
	if snapshot.Version != AbyssEscrowSnapshotVersion {
		return fmt.Errorf("unsupported Abyss escrow snapshot version %d", snapshot.Version)
	}
	if snapshot.CreatedAt.IsZero() {
		return errors.New("abyss escrow snapshot has no creation time")
	}
	providedChecksum := snapshot.Checksum
	if err := normalizeAbyssSnapshotTables(snapshot); err != nil {
		return err
	}
	checksum := abyssEscrowSnapshotChecksum(*snapshot)
	if providedChecksum == "" || providedChecksum != checksum {
		return errors.New("abyss escrow snapshot checksum mismatch")
	}

	actualCounts, err := countAbyssSnapshotRows(*snapshot)
	if err != nil {
		return err
	}
	if snapshot.Counts != actualCounts {
		return fmt.Errorf("abyss escrow snapshot row counts do not match payload: recorded=%+v actual=%+v", snapshot.Counts, actualCounts)
	}
	return nil
}

func sealAbyssEscrowSnapshot(snapshot *AbyssEscrowSnapshot) error {
	if err := normalizeAbyssSnapshotTables(snapshot); err != nil {
		return err
	}
	counts, err := countAbyssSnapshotRows(*snapshot)
	if err != nil {
		return err
	}
	snapshot.Counts = counts
	snapshot.Checksum = abyssEscrowSnapshotChecksum(*snapshot)
	return nil
}

func normalizeAbyssSnapshotTables(snapshot *AbyssEscrowSnapshot) error {
	for _, table := range []struct {
		name   string
		target *json.RawMessage
	}{
		{name: "abyss_active", target: &snapshot.Active},
		{name: "abyss_escrow_loot", target: &snapshot.Loot},
		{name: "abyss_combat_sessions", target: &snapshot.Sessions},
		{name: "abyss_combat_members", target: &snapshot.Members},
	} {
		if len(*table.target) == 0 {
			*table.target = json.RawMessage("[]")
		}
		var compact bytes.Buffer
		if err := json.Compact(&compact, *table.target); err != nil {
			return fmt.Errorf("decoding %s snapshot JSON: %w", table.name, err)
		}
		if first := compact.Bytes()[0]; first != '[' {
			return fmt.Errorf("%s snapshot payload is not an array", table.name)
		}
		*table.target = append(json.RawMessage(nil), compact.Bytes()...)
	}
	return nil
}

func countAbyssSnapshotRows(snapshot AbyssEscrowSnapshot) (AbyssEscrowSnapshotCounts, error) {
	counts := AbyssEscrowSnapshotCounts{}
	for _, table := range []struct {
		name   string
		raw    json.RawMessage
		target *int
	}{
		{name: "abyss_active", raw: snapshot.Active, target: &counts.Active},
		{name: "abyss_escrow_loot", raw: snapshot.Loot, target: &counts.Loot},
		{name: "abyss_combat_sessions", raw: snapshot.Sessions, target: &counts.Sessions},
		{name: "abyss_combat_members", raw: snapshot.Members, target: &counts.Members},
	} {
		var rows []json.RawMessage
		if err := decodeAbyssSnapshotTable(table.name, table.raw, &rows); err != nil {
			return AbyssEscrowSnapshotCounts{}, err
		}
		*table.target = len(rows)
	}
	return counts, nil
}

func decodeAbyssSnapshotTable(name string, raw json.RawMessage, target any) error {
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("decoding %s snapshot rows: %w", name, err)
	}
	return nil
}

func abyssEscrowSnapshotChecksum(snapshot AbyssEscrowSnapshot) string {
	hash := sha256.New()
	for _, table := range []struct {
		name string
		raw  json.RawMessage
	}{
		{name: "abyss_active", raw: snapshot.Active},
		{name: "abyss_escrow_loot", raw: snapshot.Loot},
		{name: "abyss_combat_sessions", raw: snapshot.Sessions},
		{name: "abyss_combat_members", raw: snapshot.Members},
	} {
		_, _ = hash.Write([]byte(table.name))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(table.raw)
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func jsonObject(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) >= 2 && trimmed[0] == '{' && json.Valid(trimmed)
}
