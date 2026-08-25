package bot

import (
	"fmt"
	"net/http"
	"time"
)

type abyssForgeUIState struct {
	TemperFailStacks       int  `json:"temper_fail_stacks"`
	TemperPityBonusPct     int  `json:"temper_pity_bonus_pct"`
	UndoAvailable          bool `json:"undo_available"`
	UndoUsedToday          bool `json:"undo_used_today"`
	Rep                    int  `json:"rep"`
	DiscountPct            int  `json:"discount_pct"`
	AccountDiscountPct     int  `json:"account_discount_pct"`
	NextDiscountRep        int  `json:"next_discount_rep"`
	RepToNextDiscount      int  `json:"rep_to_next_discount"`
	HappyHour              bool `json:"happy_hour"`
	HappyHourStartsInSecs  int  `json:"happy_hour_starts_in_seconds"`
	HappyHourEndsInSeconds int  `json:"happy_hour_ends_in_seconds"`
}

func abyssForgeHappyHourCountdown(now time.Time) (active bool, startsIn, endsIn int) {
	now = now.UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	if now.Before(start) {
		return false, max(int(start.Sub(now).Seconds()), 0), 0
	}
	if now.Before(end) {
		return true, 0, max(int(end.Sub(now).Seconds()), 0)
	}
	start = start.Add(24 * time.Hour)
	return false, max(int(start.Sub(now).Seconds()), 0), 0
}

func (b *Bot) loadAbyssForgeUIState(uid string, now time.Time) (abyssForgeUIState, error) {
	var state abyssForgeUIState
	var hasUndoSnapshot bool
	if err := b.DB.QueryRow(`SELECT temper_fail_stacks, forge_undo IS NOT NULL,
		COALESCE(forge_undo_date = CURRENT_DATE, FALSE), forge_rep
		FROM users WHERE client_uid=$1`, uid).Scan(
		&state.TemperFailStacks,
		&hasUndoSnapshot,
		&state.UndoUsedToday,
		&state.Rep,
	); err != nil {
		return state, fmt.Errorf("load Abyss forge UI state: %w", err)
	}
	state.TemperPityBonusPct = min(max(state.TemperFailStacks, 0)*5, 100)
	state.UndoAvailable = hasUndoSnapshot && !state.UndoUsedToday
	state.DiscountPct = forgeDiscountPct(state.Rep)
	if state.DiscountPct < 20 {
		state.NextDiscountRep = (max(state.Rep, 0)/25 + 1) * 25
		state.RepToNextDiscount = max(state.NextDiscountRep-state.Rep, 0)
	} else {
		state.NextDiscountRep = state.Rep
	}
	state.HappyHour, state.HappyHourStartsInSecs, state.HappyHourEndsInSeconds = abyssForgeHappyHourCountdown(now)
	state.AccountDiscountPct = state.DiscountPct
	if state.HappyHour {
		state.AccountDiscountPct += 20
	}
	state.AccountDiscountPct = min(state.AccountDiscountPct, 40)
	return state, nil
}

func (s *WebServer) handleAbyssForgeUIState(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	state, err := s.bot.loadAbyssForgeUIState(uid, time.Now())
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "state": state})
}
