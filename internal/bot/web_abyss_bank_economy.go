package bot

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

type abyssTaxLeaderView struct {
	Nickname string
	Tax      int64
	Badge    string
}

func abyssLoanLimit(escrow, borrowed int64) int64 {
	original := max(int64(0), escrow) + max(int64(0), borrowed)
	return max(int64(0), original/2-borrowed)
}

func abyssLoanFee(principal int64) int64 {
	if principal <= 0 {
		return 0
	}
	return (principal + 9) / 10
}

func abyssEconomyWeek(now time.Time) string {
	year, week := now.UTC().ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

func (b *Bot) currentAbyssLoanFee(uid string) int64 {
	var fee int64
	_ = b.DB.QueryRow("SELECT economy_loan_fee FROM abyss_active WHERE client_uid=$1", uid).Scan(&fee)
	return max(int64(0), fee)
}

func (s *WebServer) handleAbyssEconomyLoan(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var req struct {
		Amount int64 `json:"amount"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var escrow, borrowed int64
	var downed bool
	if err := tx.QueryRow(`SELECT escrow,economy_loan_principal,downed FROM abyss_active
		WHERE client_uid=$1 FOR UPDATE`, uid).Scan(&escrow, &borrowed, &downed); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "no active run"})
		return
	}
	if downed {
		writeJSON(w, map[string]any{"ok": false, "error": "loans are unavailable while downed"})
		return
	}
	available := abyssLoanLimit(escrow, borrowed)
	amount := req.Amount
	if amount <= 0 {
		amount = available
	}
	if amount <= 0 || amount > available {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("loan amount exceeds the remaining 50%% cache limit (%d)", available)})
		return
	}
	fee := abyssLoanFee(amount)
	if _, err := tx.Exec(`UPDATE abyss_active SET escrow=escrow-$1,
		economy_loan_principal=economy_loan_principal+$1,economy_loan_fee=economy_loan_fee+$2,last_action_at=NOW()
		WHERE client_uid=$3`, amount, fee, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec("UPDATE users SET gold=gold+$1 WHERE client_uid=$2", amount, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "borrowed": amount, "fee_due": fee, "escrow": escrow - amount,
		"gold": s.bot.abyssGold(uid), "msg": fmt.Sprintf("Cache loan paid %dg now; %dg fee is due at the next bank.", amount, fee)})
}

func recordAbyssTax(tx *sql.Tx, uid string, tax int64) error {
	if tax <= 0 {
		return nil
	}
	_, err := tx.Exec(`INSERT INTO abyss_economy_events (client_uid,kind,message,amount)
		VALUES ($1,'tax_paid',$2,$3)`, uid, fmt.Sprintf("Abyss tax paid: %dg.", tax), tax)
	return err
}

func (b *Bot) abyssTaxLeaders(limit int) []abyssTaxLeaderView {
	if limit < 1 || limit > 25 {
		limit = 5
	}
	rows, err := b.DB.Query(`SELECT COALESCE(NULLIF(u.nickname,''),'Adventurer'),SUM(e.amount)
		FROM abyss_economy_events e JOIN users u ON u.client_uid=e.client_uid
		WHERE e.kind='tax_paid' AND e.created_at >= date_trunc('month',NOW())
		GROUP BY e.client_uid,u.nickname ORDER BY SUM(e.amount) DESC LIMIT $1`, limit)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []abyssTaxLeaderView
	for rows.Next() {
		var view abyssTaxLeaderView
		if rows.Scan(&view.Nickname, &view.Tax) == nil {
			if len(out) == 0 {
				view.Badge = "Tax Titan"
			}
			out = append(out, view)
		}
	}
	return out
}

func (s *WebServer) handleAbyssTaxRebate(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var bankedToday bool
	if err := s.bot.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM abyss_runs
		WHERE client_uid=$1 AND victory=TRUE AND created_at>=CURRENT_DATE)`, uid).Scan(&bankedToday); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if bankedToday {
		writeJSON(w, map[string]any{"ok": false, "error": "rebates require a bank-free UTC day"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`INSERT INTO abyss_economy_profiles (client_uid) VALUES ($1) ON CONFLICT DO NOTHING`, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	week := abyssEconomyWeek(time.Now())
	res, err := tx.Exec(`UPDATE abyss_economy_profiles SET tax_rebate_week=$2
		WHERE client_uid=$1 AND tax_rebate_week IS DISTINCT FROM $2`, uid, week)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "weekly tax rebate already claimed"})
		return
	}
	var paid int64
	if err := tx.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM abyss_economy_events
		WHERE client_uid=$1 AND kind='tax_paid' AND created_at>=date_trunc('week',NOW())`, uid).Scan(&paid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	rebate := paid / 10
	if rebate <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "no tax paid this week"})
		return
	}
	if _, err := tx.Exec("UPDATE users SET gold=gold+$1 WHERE client_uid=$2", rebate, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "rebate": rebate, "gold": s.bot.abyssGold(uid),
		"msg": fmt.Sprintf("Bank-free day rebate: %dg (10%% of %dg weekly tax).", rebate, paid)})
}

func (b *Bot) splitAbyssJackpot(uid, helperUID string, jackpot int64) int64 {
	if helperUID == "" || helperUID == uid || jackpot <= 0 {
		return 0
	}
	split := abyssJackpotHelperShare(jackpot)
	if split <= 0 {
		return 0
	}
	tx, err := b.DB.Begin()
	if err != nil {
		return 0
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.Exec("UPDATE users SET gold=gold-$1 WHERE client_uid=$2 AND gold >= $1", split, uid)
	if err != nil {
		return 0
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return 0
	}
	if _, err := tx.Exec("UPDATE users SET gold=LEAST(9223372036854775807::numeric, gold::numeric+$1)::bigint WHERE client_uid=$2", split, helperUID); err != nil {
		return 0
	}
	if _, err := tx.Exec(`INSERT INTO abyss_economy_events (client_uid,kind,message,amount)
		VALUES ($1,'jackpot_split',$2,$3)`, helperUID, fmt.Sprintf("Co-op jackpot share received: %dg.", split), split); err != nil {
		return 0
	}
	if err := tx.Commit(); err != nil {
		return 0
	}
	return split
}

func abyssJackpotHelperShare(jackpot int64) int64 {
	return max(int64(0), jackpot) / 10
}

func abyssBountyContinues(claimedYesterday, claimedTwoDaysAgo, insuranceAvailable bool) (continues, useInsurance bool) {
	if claimedYesterday {
		return true, false
	}
	return claimedTwoDaysAgo && insuranceAvailable, claimedTwoDaysAgo && insuranceAvailable
}

func (b *Bot) bountyInsuranceAvailable(uid string, day time.Time) bool {
	week := abyssEconomyWeek(day)
	var available bool
	_ = b.DB.QueryRow(`SELECT NOT EXISTS(SELECT 1 FROM abyss_economy_profiles
		WHERE client_uid=$1 AND bounty_insurance_week=$2 AND bounty_insurance_used=TRUE)`, uid, week).Scan(&available)
	return available
}

func useBountyInsurance(tx *sql.Tx, uid string, day time.Time) error {
	week := abyssEconomyWeek(day)
	_, err := tx.Exec(`INSERT INTO abyss_economy_profiles (client_uid,bounty_insurance_week,bounty_insurance_used)
		VALUES ($1,$2,TRUE) ON CONFLICT (client_uid) DO UPDATE SET
		bounty_insurance_week=$2,bounty_insurance_used=TRUE`, uid, week)
	return err
}
