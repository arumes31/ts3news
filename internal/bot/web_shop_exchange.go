package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"
)

type shopXPExchangeState struct {
	Week  string `json:"week"`
	Count int64  `json:"count"`
}

func shopXPExchangeWeek(now time.Time) string {
	year, week := now.UTC().ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

func shopXPExchangeRate(count int64) int64 {
	const step = goldPerXP / 10
	return goldPerXP + min(max(count, 0), (math.MaxInt64-goldPerXP)/step)*step
}

func shopXPExchangePurchases(raw, week string) (int64, error) {
	var state shopXPExchangeState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return 0, err
	}
	if state.Count < 0 || state.Week == "" {
		return 0, fmt.Errorf("invalid exchange history")
	}
	if state.Week != week {
		return 0, nil
	}
	return state.Count, nil
}

func shopXPExchangeKey(uid string) string { return "shop_xp_purchases_" + uid }

type shopExchangeReader interface {
	QueryRow(string, ...any) *sql.Row
}

// The purchase handler holds the wallet lock while reading and saving this
// history, so simultaneous requests cannot reuse the same weekly price.
func loadShopXPExchangeCount(reader shopExchangeReader, uid, week string) (int64, error) {
	var raw string
	err := reader.QueryRow("SELECT value FROM app_meta WHERE key=$1", shopXPExchangeKey(uid)).Scan(&raw)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return shopXPExchangePurchases(raw, week)
}
