package bot

import (
	"encoding/json"
	"time"
)

type abyssHistoryRow struct {
	RunID         int64    `json:"run_id"`
	Depth         int      `json:"depth"`
	Gold          int64    `json:"gold"`
	Victory       bool     `json:"victory"`
	Tier          string   `json:"tier"`
	Hardcore      bool     `json:"hardcore"`
	EndReason     string   `json:"end_reason"`
	LootCount     int      `json:"loot_count"`
	Loot          []string `json:"loot"`
	LootTruncated int      `json:"loot_truncated"`
	When          string   `json:"when"`
	AtUnix        int64    `json:"at_unix"`
	DurationMS    int64    `json:"duration_ms"`
	FloorsCleared int      `json:"floors_cleared"`
	AuditHash     string   `json:"audit_hash,omitempty"`
	ReplayReady   bool     `json:"replay_ready"`
}

func abyssRunDurationMS(run abyssRun) int64 {
	if run.StartedAt.IsZero() {
		return 0
	}
	return max(time.Since(run.StartedAt).Milliseconds(), 0)
}

func (b *Bot) abyssHistory(uid string, limit int) []abyssHistoryRow {
	limit = min(max(limit, 1), 50)
	rows, err := b.DB.Query(
		`SELECT id, depth, gold_banked, victory, COALESCE(tier, 'normal'), hardcore,
		        end_reason, loot_count, loot_summary, created_at, duration_ms, floors_cleared,
		        audit_hash, audit_data
		   FROM abyss_runs WHERE client_uid=$1 ORDER BY id DESC LIMIT $2`,
		uid, limit)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	out := make([]abyssHistoryRow, 0, limit)
	for rows.Next() {
		var h abyssHistoryRow
		var lootJSON []byte
		var auditJSON []byte
		var when time.Time
		if err := rows.Scan(
			&h.RunID, &h.Depth, &h.Gold, &h.Victory, &h.Tier, &h.Hardcore, &h.EndReason,
			&h.LootCount, &lootJSON, &when, &h.DurationMS, &h.FloorsCleared,
			&h.AuditHash, &auditJSON,
		); err != nil {
			continue
		}
		_ = json.Unmarshal(lootJSON, &h.Loot)
		var audit abyssCompetitionAudit
		if h.AuditHash != "" && json.Unmarshal(auditJSON, &audit) == nil {
			h.ReplayReady = len(audit.Floors) > 0
		}
		h.LootTruncated = max(0, h.LootCount-len(h.Loot))
		h.When = when.Format("Jan 2 15:04")
		h.AtUnix = when.Unix()
		out = append(out, h)
	}
	return out
}
