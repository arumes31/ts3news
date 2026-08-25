package bot

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	abyssFranticFeePct          = 5
	abyssFranticHPThresholdPct  = 15
	abyssBankSafeWordThreshold  = int64(1_000_000)
	abyssRestShrinkRatePct      = 70
	abyssRestShrinkGoldPerToken = int64(100_000)
)

func abyssGreedyInterestRate(base float64, depth int) float64 {
	return base + float64(abyssGreedyGripStacks(depth))*0.02
}

func abyssIdleDangerPct(lastAction time.Time, now time.Time) int {
	if lastAction.IsZero() || !now.After(lastAction) {
		return 0
	}
	minutes := int(now.Sub(lastAction) / time.Minute)
	return min(max(minutes, 0), 50)
}

func abyssBankNeedsSafeWord(payout int64, confirmationEnabled bool) bool {
	return confirmationEnabled && payout > abyssBankSafeWordThreshold
}

// handleAbyssRestCacheShrink converts exactly half of a rest-floor cache at
// 70% of the normal gold-to-token rate. One conversion is allowed per rest
// floor, and cache + token mutations commit atomically.
func (s *WebServer) handleAbyssRestCacheShrink(w http.ResponseWriter, r *http.Request, uid string) {
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
	if !run.Active || run.Downed || run.FloorType != "rest" {
		writeJSON(w, map[string]any{"ok": false, "error": "cache shrink is available only on a live rest floor"})
		return
	}
	flags := s.bot.loadRunFlags(uid)
	if flags["cache_shrink_depth"] == int64(run.Depth) {
		writeJSON(w, map[string]any{"ok": false, "error": "cache already shrunk on this rest floor"})
		return
	}
	converted, tokens := abyssRestCacheConversion(run.Escrow)
	if tokens < 1 {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("need at least %dg in the cache to mint one token", abyssRestShrinkGoldPerToken*200/abyssRestShrinkRatePct)})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var remaining int64
	if err := tx.QueryRow(
		`UPDATE abyss_active SET escrow=escrow-$1, last_action_at=NOW()
		  WHERE client_uid=$2 AND floor_type='rest' AND escrow=$3 RETURNING escrow`,
		converted, uid, run.Escrow,
	).Scan(&remaining); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "run changed; refresh and try again"})
		return
	}
	if _, err := tx.Exec("UPDATE users SET abyss_tokens=abyss_tokens+$1 WHERE client_uid=$2", tokens, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	flags["cache_shrink_depth"] = int64(run.Depth)
	if err := saveRunFlags(tx, uid, flags); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{
		"ok": true, "converted": converted, "escrow": remaining,
		"tokens_gained": tokens, "tokens": s.bot.abyssTokens(uid),
		"msg": fmt.Sprintf("🜲 The sanctuary compresses %dg of cache into %d token(s).", converted, tokens),
	})
}

func normalizeAbyssSafeWord(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

// handleAbyssDownedTimeout lets the downed screen enforce its deadline even
// when the player makes no further choice. The actual forfeit remains guarded
// by the per-player lock and the server timestamp.
func (s *WebServer) handleAbyssDownedTimeout(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	run := s.bot.loadAbyssRun(uid)
	if s.autoConcedeIfTimedOut(w, uid, run) {
		return
	}
	remaining := 0
	if run.Active && run.Downed && !run.LastActionAt.IsZero() {
		remaining = max(int(time.Until(run.LastActionAt.Add(abyssDownedTimeout)).Seconds()), 0)
	}
	writeJSON(w, map[string]any{"ok": true, "auto_conceded": false, "remaining_seconds": remaining})
}
