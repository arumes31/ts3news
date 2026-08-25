package bot

import (
	"encoding/json"
	"time"
)

type abyssHistoryRow struct {
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
}

func (b *Bot) abyssHistory(uid string, limit int) []abyssHistoryRow {
	limit = min(max(limit, 1), 50)
	rows, err := b.DB.Query(
		`SELECT depth, gold_banked, victory, COALESCE(tier, 'normal'), hardcore,
		        end_reason, loot_count, loot_summary, created_at
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
		var when time.Time
		if err := rows.Scan(
			&h.Depth, &h.Gold, &h.Victory, &h.Tier, &h.Hardcore, &h.EndReason,
			&h.LootCount, &lootJSON, &when,
		); err != nil {
			continue
		}
		_ = json.Unmarshal(lootJSON, &h.Loot)
		h.LootTruncated = max(0, h.LootCount-len(h.Loot))
		h.When = when.Format("Jan 2 15:04")
		h.AtUnix = when.Unix()
		out = append(out, h)
	}
	return out
}
