package bot

// Generic Abyss talents: the Deep-Delver extension and the per-spec sub-trees.
// Levels are stored generically as key→level JSON in app_meta (no per-node DB
// column), and each allocated level's effect is folded into treeBonusFor so it
// rides the same live combat/economy pipeline as the skill web. See
// internal/content/abyss_talents.go for the node definitions.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"ts3news/internal/content"
)

func abyssTalentKey(uid string) string { return "abyss_talents_" + uid }

// talentTokenCost is the Abyss-token cost of upgrading a talent from level to
// level+1 (level is 0-based, so the first level costs 10). Shared by the spend,
// the generic refund and the legacy talent reset so the pricing can't drift.
func talentTokenCost(level int) int64 { return int64(level+1) * 10 }

func abyssTalentEffectiveInt(level int) int {
	return int(content.TalentEffectiveLevel(level) + 0.5)
}

func persistAbyssTalentUpgrade(tx *sql.Tx, uid string, cost int64, levels map[string]int) (bool, error) {
	res, err := tx.Exec(
		"UPDATE users SET abyss_tokens = abyss_tokens - $1 WHERE client_uid=$2 AND abyss_tokens >= $1", cost, uid)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	if err != nil || rows == 0 {
		return false, err
	}
	data, err := json.Marshal(levels)
	if err != nil {
		return false, err
	}
	_, err = tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssTalentKey(uid), string(data))
	return err == nil, err
}

// loadAbyssTalentLevels returns the player's allocated generic-talent levels.
func (b *Bot) loadAbyssTalentLevels(uid string) map[string]int {
	out := map[string]int{}
	var js string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssTalentKey(uid)).Scan(&js)
	if js != "" {
		if err := json.Unmarshal([]byte(js), &out); err != nil {
			log.Printf("abyss talent levels corrupt for %s: %v", uid, err)
		}
	}
	// Corrupt negative levels are clamped here (the consumers only clamp the top).
	for k, lvl := range out {
		if lvl < 0 {
			out[k] = 0
		}
	}
	return out
}

// abyssTalentBonus sums the player's allocated generic talents into one bonus
// block (Deep-Delver + the active spec's sub-tree). Folded into treeBonusFor.
func (b *Bot) abyssTalentBonus(uid string) content.TreeBonus {
	return content.TalentBonus(b.loadAbyssTalentLevels(uid), b.abyssSpec(uid))
}

// talentLevelOf reads a prerequisite's current level, transparently spanning the
// legacy per-column Deep-Delver nodes and the generic key→level store, so a
// generic node can hang off a legacy leaf (e.g. Scavenger) as its parent.
func (b *Bot) talentLevelOf(uid, key string) int {
	if col, ok := abyssUpgradeCols[key]; ok { // col is whitelisted → safe to interpolate
		var lvl int
		_ = b.DB.QueryRow("SELECT "+col+" FROM users WHERE client_uid=$1", uid).Scan(&lvl)
		return lvl
	}
	return b.loadAbyssTalentLevels(uid)[key]
}

// handleAbyssTalentUpgrade spends tokens on a generic talent level. The caller
// (handleAbyssUpgrade) already holds the per-uid abyss lock, so the token debit
// and the app_meta level bump can't race for the same player.
func (s *WebServer) handleAbyssTalentUpgrade(w http.ResponseWriter, uid string, t content.Talent) {
	levels := s.bot.loadAbyssTalentLevels(uid)
	level := levels[t.Key]
	if level >= content.TalentMaxLevel {
		writeJSON(w, map[string]any{"ok": false, "error": "maxed"})
		return
	}
	if t.Spec != "" && s.bot.abyssSpec(uid) != t.Spec {
		writeJSON(w, map[string]any{"ok": false, "error": "locked — activate the matching specialization first"})
		return
	}
	if t.GateDepth > 0 && s.bot.loadAbyssStats(uid).BestDepth < t.GateDepth {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("locked — reach depth %d first", t.GateDepth)})
		return
	}
	if t.Parent != "" && s.bot.talentLevelOf(uid, t.Parent) < 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "locked — upgrade the prerequisite first"})
		return
	}
	cost := talentTokenCost(level)
	// Guarded debit: only proceeds if the player still has the tokens (matches the
	// legacy Deep-Delver spend). RowsAffected==0 means someone else spent first.
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	levels[t.Key] = level + 1
	spent, err := persistAbyssTalentUpgrade(tx, uid, cost, levels)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if !spent {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough tokens"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "node": t.Key, "level": level + 1, "tokens": s.bot.abyssTokens(uid)})
}

// abyssTalentRefund totals the tokens sunk into every generic talent (used by the
// talent reset so the generic nodes refund alongside the legacy columns).
func abyssTalentRefund(levels map[string]int) int64 {
	var refund int64
	for key, lvl := range levels {
		if _, ok := content.TalentByKey(key); !ok {
			continue // ignore stale keys from a removed node
		}
		if lvl > content.TalentMaxLevel {
			lvl = content.TalentMaxLevel
		}
		for l := 1; l <= lvl; l++ {
			refund += talentTokenCost(l - 1)
		}
	}
	return refund
}
