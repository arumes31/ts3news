package bot

import (
	"database/sql"
	"log"
	"time"
)

// Jackpot increment rates (per 100 gold bet)
const jackpotRate = 0.01 // 1% of all bets go to jackpots

type vipTier struct {
	Name    string
	Points  int
	Bonus   int // % gold bonus
	Rebate  int // % loss-back
}

var vipTiers = []vipTier{
	{"None", 0, 0, 0},
	{"Bronze", 10000, 2, 0},
	{"Silver", 50000, 5, 5},
	{"Gold", 250000, 10, 10},
	{"Platinum", 1000000, 15, 15},
	{"Diamond", 5000000, 25, 20},
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

func (b *Bot) addVIPPoints(uid string, points int) {
	_, _ = b.DB.Exec("UPDATE users SET vip_points = vip_points + $1 WHERE client_uid=$2", points, uid)
}

func (b *Bot) getJackpot(game string) int64 {
	var amt int64
	err := b.DB.QueryRow("SELECT amount FROM arcade_jackpots WHERE game_key=$1", game).Scan(&amt)
	if err != nil {
		return 10000 // default
	}
	return amt
}

func (b *Bot) incrementJackpot(game string, bet int64) {
	inc := int64(float64(bet) * jackpotRate)
	if inc < 1 {
		inc = 1
	}
	_, _ = b.DB.Exec("UPDATE arcade_jackpots SET amount = amount + $1, updated_at = NOW() WHERE game_key=$1", inc, game)
}

func (b *Bot) claimJackpot(uid string, game string) int64 {
	var amt int64
	err := b.DB.QueryRow("UPDATE arcade_jackpots SET amount = 10000, updated_at = NOW() WHERE game_key=$1 RETURNING amount", game).Scan(&amt)
	if err != nil {
		log.Printf("jackpot: claim failed: %v", err)
		return 0
	}
	// Note: the UPDATE .. RETURNING above actually returns the NEW amount (10000).
	// We need the OLD amount.
	
	tx, err := b.DB.Begin()
	if err != nil { return 0 }
	defer func() { _ = tx.Rollback() }()

	_ = tx.QueryRow("SELECT amount FROM arcade_jackpots WHERE game_key=$1", game).Scan(&amt)
	_, _ = tx.Exec("UPDATE arcade_jackpots SET amount = 10000, updated_at = NOW() WHERE game_key=$1", game)
	_, _ = tx.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid=$2", amt, uid)
	
	if err := tx.Commit(); err != nil { return 0 }
	return amt
}

func (b *Bot) canSpinDaily(uid string) bool {
	var last sql.NullTime
	_ = b.DB.QueryRow("SELECT last_daily_spin FROM users WHERE client_uid=$1", uid).Scan(&last)
	if !last.Valid {
		return true
	}
	return time.Since(last.Time) > 24*time.Hour
}

func (b *Bot) recordDailySpin(uid string) {
	_, _ = b.DB.Exec("UPDATE users SET last_daily_spin = NOW() WHERE client_uid=$2", uid)
}
