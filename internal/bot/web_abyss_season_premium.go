package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const abyssSeasonPremiumUnlockCost = int64(40)

func abyssSeasonPremiumEntitlementKey(campaign abyssSeasonCampaign) string {
	return "season_" + campaign.ID + "_premium"
}

func abyssSeasonPremiumCosmeticKey(campaign abyssSeasonCampaign, week int) string {
	return fmt.Sprintf("season_%s_premium_week_%02d", campaign.ID, week)
}

func abyssSeasonPremiumRewardName(campaign abyssSeasonCampaign, week int) string {
	return campaign.RewardWord + " Gilded " + abyssSeasonRewardNames[week-1]
}

func (s *WebServer) handleAbyssSeasonPremiumUnlock(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	campaign := abyssSeasonCampaignAt(time.Now().UTC())
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(r.Context(), `INSERT INTO abyss_shop_cosmetics (client_uid,cosmetic_key) VALUES ($1,$2)
		ON CONFLICT (client_uid,cosmetic_key) DO NOTHING`, uid, abyssSeasonPremiumEntitlementKey(campaign))
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	changed, err := result.RowsAffected()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	response := map[string]any{"ok": true, "unlocked": true, "already_unlocked": changed == 0}
	if changed > 0 {
		var tokens int64
		err = tx.QueryRowContext(r.Context(), `UPDATE users SET abyss_tokens=abyss_tokens-$1
			WHERE client_uid=$2 AND abyss_tokens >= $1 RETURNING abyss_tokens`, abyssSeasonPremiumUnlockCost, uid).Scan(&tokens)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, map[string]any{"ok": false, "error": "not enough tokens"})
			} else {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
			}
			return
		}
		response["tokens"] = tokens
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, response)
}

func (s *WebServer) handleAbyssSeasonPremiumClaim(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Week int `json:"week"`
	}
	if readJSON(r, &req) != nil || req.Week < 1 || req.Week > abyssSeasonWeeks {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid season week"})
		return
	}
	campaign := abyssSeasonCampaignAt(time.Now().UTC())
	if req.Week > campaign.CurrentWeek {
		writeJSON(w, map[string]any{"ok": false, "error": "season week is not available yet"})
		return
	}
	progress, err := s.bot.abyssSeasonProgress(r.Context(), uid, campaign)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if progress[req.Week-1] < abyssSeasonWeekGoals[req.Week-1] {
		writeJSON(w, map[string]any{"ok": false, "error": "weekly journey objective is not complete"})
		return
	}
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var entitled bool
	if err := tx.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM abyss_shop_cosmetics
		WHERE client_uid=$1 AND cosmetic_key=$2)`, uid, abyssSeasonPremiumEntitlementKey(campaign)).Scan(&entitled); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if !entitled {
		writeJSON(w, map[string]any{"ok": false, "error": "premium lane is locked"})
		return
	}
	key := abyssSeasonPremiumCosmeticKey(campaign, req.Week)
	result, err := tx.ExecContext(r.Context(), `INSERT INTO abyss_shop_cosmetics (client_uid,cosmetic_key) VALUES ($1,$2)
		ON CONFLICT (client_uid,cosmetic_key) DO NOTHING`, uid, key)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	changed, err := result.RowsAffected()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{
		"ok": true, "week": req.Week, "name": abyssSeasonPremiumRewardName(campaign, req.Week),
		"claimed": true, "already_owned": changed == 0,
	})
}
