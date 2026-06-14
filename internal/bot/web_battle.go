package bot

import "time"

// battleHistoryRow is one row of the per-user auto-battler history shown on the
// battle page. Fights are recorded by the TFT combat handler (see web_tft.go).
type battleHistoryRow struct {
	MobName string
	Victory bool
	GoldWon int64
	GearWon string
	When    string
}

func (b *Bot) battleHistory(uid string, limit int) []battleHistoryRow {
	rows, err := b.DB.Query(
		"SELECT mob_name, victory, gold_won, COALESCE(gear_won,''), fought_at FROM battle_history WHERE client_uid=$1 ORDER BY fought_at DESC LIMIT $2",
		uid, limit)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []battleHistoryRow
	for rows.Next() {
		var row battleHistoryRow
		var t time.Time
		if err := rows.Scan(&row.MobName, &row.Victory, &row.GoldWon, &row.GearWon, &t); err != nil {
			continue
		}
		row.When = t.Format("Jan 02 15:04")
		out = append(out, row)
	}
	return out
}
