package bot

import (
	"database/sql"
	"encoding/json"
	"math/rand/v2"
	"net/http"
)

const (
	abyssRunFlagDoubleBonus      = "double_bonus"
	abyssRunFlagDoubleBonusDepth = "double_bonus_depth"
	abyssPartialBankFeePct       = 10
)

type abyssPartialBankQuote struct {
	Escrow    int64
	Gross     int64
	Fee       int64
	Payout    int64
	Remaining int64
}

func quoteAbyssPartialBank(escrow int64, multiplier float64, percent int) (abyssPartialBankQuote, bool) {
	if escrow <= 0 || (percent != 25 && percent != 50) {
		return abyssPartialBankQuote{}, false
	}
	share := escrow * int64(percent) / 100
	if share <= 0 {
		return abyssPartialBankQuote{}, false
	}
	gross := int64(float64(share) * multiplier)
	fee := gross * abyssPartialBankFeePct / 100
	return abyssPartialBankQuote{
		Escrow:    share,
		Gross:     gross,
		Fee:       fee,
		Payout:    gross - fee,
		Remaining: escrow - share,
	}, true
}

func resolveAbyssDoubleBonus(escrow, bonus int64, won bool) int64 {
	if bonus <= 0 {
		return escrow
	}
	if won {
		return escrow + bonus
	}
	return max(escrow-bonus, 0)
}

func (b *Bot) setPendingAbyssDoubleBonus(uid string, depth int, bonus int64) error {
	flags := b.loadRunFlags(uid)
	flags[abyssRunFlagDoubleBonus] = max(bonus, 0)
	flags[abyssRunFlagDoubleBonusDepth] = int64(depth)
	return b.saveRunFlags(uid, flags)
}

func pendingAbyssDoubleBonus(flags map[string]int64, depth int) int64 {
	if flags[abyssRunFlagDoubleBonusDepth] != int64(depth) {
		return 0
	}
	return max(flags[abyssRunFlagDoubleBonus], 0)
}

func consumePendingAbyssDoubleBonus(tx *sql.Tx, uid string, flags map[string]int64) error {
	flags[abyssRunFlagDoubleBonus] = 0
	flags[abyssRunFlagDoubleBonusDepth] = 0
	encoded, err := json.Marshal(flags)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssRunFlagsKey(uid), string(encoded))
	return err
}

// handleAbyssDoubleBonus resolves the optional one-shot coin flip attached to
// the most recently cleared floor. The escrow mutation and consumed offer are
// committed together, preventing refreshes or repeated requests from rerolling.
func (s *WebServer) handleAbyssDoubleBonus(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}

	run := s.bot.loadAbyssRun(uid)
	if !run.Active || run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "no live run"})
		return
	}
	flags := s.bot.loadRunFlags(uid)
	bonus := pendingAbyssDoubleBonus(flags, run.Depth)
	if bonus <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "no floor bonus gamble is available"})
		return
	}

	won := rand.IntN(2) == 0 // #nosec G404 -- intentional gameplay coin flip
	newEscrow := resolveAbyssDoubleBonus(run.Escrow, bonus, won)
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec("UPDATE abyss_active SET escrow=$1, last_action_at=NOW() WHERE client_uid=$2", newEscrow, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := consumePendingAbyssDoubleBonus(tx, uid, flags); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{
		"ok": true, "won": won, "bonus": bonus, "escrow": newEscrow, "depth": run.Depth,
	})
}
