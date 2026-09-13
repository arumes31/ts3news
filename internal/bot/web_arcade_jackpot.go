package bot

import (
	"database/sql"
	"time"
)

// Jackpot increment rates (per 100 gold bet)
const jackpotRate = 0.01 // 1% of all bets go to jackpots

type vipTier struct {
	Name   string
	Points int
	Bonus  int // % gold bonus
	Rebate int // % loss-back
}

var vipTiers = []vipTier{
	{"None", 0, 0, 0},
	{"Bronze", 10000, 2, 0},
	{"Silver", 50000, 5, 0},
	{"Gold", 250000, 10, 0},
	{"Platinum", 1000000, 15, 1},
	{"Diamond", 5000000, 25, 1},
}

func (b *Bot) getVIP(uid string) (vipTier, int) {
	var p int
	_ = b.DB.QueryRow("SELECT vip_points FROM users WHERE client_uid=$1", uid).Scan(&p)

	current := vipTiers[0]
	for _, t := range vipTiers {
		if p >= t.Points {
			current = t
		}
	}
	return current, p
}

func (b *Bot) getJackpot(game string) int64 {
	var amt int64
	err := b.DB.QueryRow("SELECT amount FROM arcade_jackpots WHERE game_key=$1", game).Scan(&amt)
	if err != nil {
		return 0 // Only funded balances can be awarded.
	}
	return amt
}

func (b *Bot) incrementJackpot(game string, bet int64) {
	inc := int64(float64(bet) * jackpotRate)
	if inc <= 0 {
		return
	}
	_, _ = b.DB.Exec("UPDATE arcade_jackpots SET amount = amount + $1, updated_at = NOW() WHERE game_key=$2", inc, game)
}

func (b *Bot) claimJackpot(uid string, game string) int64 {
	tx, err := b.DB.Begin()
	if err != nil {
		return 0
	}
	defer func() { _ = tx.Rollback() }()
	// Match arcade settlement lock order: account before shared pot.
	var gold, pool int64
	if err := tx.QueryRow("SELECT gold FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&gold); err != nil {
		return 0
	}
	if gold < 0 {
		return 0
	}
	if err := tx.QueryRow("SELECT amount FROM arcade_jackpots WHERE game_key=$1 FOR UPDATE", game).Scan(&pool); err != nil {
		return 0
	}
	amount := max(int64(0), min(pool, int64(9223372036854775807)-gold))
	if amount == 0 {
		return 0
	}
	if _, err := tx.Exec("UPDATE arcade_jackpots SET amount = amount - $1, updated_at = NOW() WHERE game_key=$2", amount, game); err != nil {
		return 0
	}
	if _, err := tx.Exec("/* economy:bot.Bot.claimJackpot */ UPDATE users SET gold = gold + $1 WHERE client_uid=$2", amount, uid); err != nil {
		return 0
	}
	if err := tx.Commit(); err != nil {
		return 0
	}
	return amount
}

func (b *Bot) attemptDailySpin(uid string) bool {
	res, err := b.DB.Exec(`/* economy:bot.Bot.attemptDailySpin */
		UPDATE users 
		SET last_daily_spin = NOW() 
		WHERE client_uid = $1 
		  AND (last_daily_spin IS NULL OR last_daily_spin < NOW() - INTERVAL '24 hours')`,
		uid)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

func (b *Bot) canSpinDaily(uid string) bool {
	var last sql.NullTime
	_ = b.DB.QueryRow("SELECT last_daily_spin FROM users WHERE client_uid=$1", uid).Scan(&last)
	if !last.Valid {
		return true
	}
	return time.Since(last.Time) > 24*time.Hour
}
