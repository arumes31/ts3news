package bot

import (
	"database/sql"
	"fmt"
)

type abyssAffixStatView struct {
	Key          string  `json:"key"`
	Label        string  `json:"label"`
	Runs         int64   `json:"runs"`
	Wins         int64   `json:"wins"`
	WinRatePct   float64 `json:"win_rate_pct"`
	AverageDepth float64 `json:"average_depth"`
}

func abyssDailyAffixFromFlags(flags map[string]int64) string {
	index := int(flags[abyssRunFlagDailyAffix]) - 1
	if index < 0 || index >= len(abyssDailyMods) {
		return ""
	}
	return abyssDailyMods[index]
}

func recordAbyssAffixRun(tx *sql.Tx, uid, affix string, depth int, victory bool) error {
	if abyssDailyAffixIndex(affix) == 0 || depth <= 0 {
		return nil
	}
	wins := 0
	if victory {
		wins = 1
	}
	if _, err := tx.Exec(
		`INSERT INTO abyss_affix_stats (client_uid, affix_key, runs, wins, total_depth)
		 VALUES ($1,$2,1,$3,$4)
		 ON CONFLICT (client_uid, affix_key) DO UPDATE SET
		   runs=abyss_affix_stats.runs+1,
		   wins=abyss_affix_stats.wins+EXCLUDED.wins,
		   total_depth=abyss_affix_stats.total_depth+EXCLUDED.total_depth,
		   updated_at=NOW()`,
		uid, affix, wins, depth,
	); err != nil {
		return fmt.Errorf("record Abyss affix run: %w", err)
	}
	return nil
}

func (b *Bot) loadAbyssAffixStats(uid string) ([]abyssAffixStatView, error) {
	rows, err := b.DB.Query(
		"SELECT affix_key, runs, wins, total_depth FROM abyss_affix_stats WHERE client_uid=$1",
		uid,
	)
	if err != nil {
		return nil, fmt.Errorf("load Abyss affix stats: %w", err)
	}
	defer func() { _ = rows.Close() }()
	stored := make(map[string]abyssAffixStatView, len(abyssDailyMods))
	for rows.Next() {
		var view abyssAffixStatView
		var totalDepth int64
		if err := rows.Scan(&view.Key, &view.Runs, &view.Wins, &totalDepth); err != nil {
			return nil, fmt.Errorf("scan Abyss affix stats: %w", err)
		}
		if abyssDailyAffixIndex(view.Key) == 0 || view.Runs <= 0 {
			continue
		}
		view.Label = abyssDailyAffixLabel(view.Key)
		view.WinRatePct = pctNumber1(float64(view.Wins) * 100 / float64(view.Runs))
		view.AverageDepth = pctNumber1(float64(totalDepth) / float64(view.Runs))
		stored[view.Key] = view
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Abyss affix stats: %w", err)
	}
	views := make([]abyssAffixStatView, 0, len(abyssDailyMods))
	for _, key := range abyssDailyMods {
		view := stored[key]
		if view.Key == "" {
			view = abyssAffixStatView{Key: key, Label: abyssDailyAffixLabel(key)}
		}
		views = append(views, view)
	}
	return views, nil
}
