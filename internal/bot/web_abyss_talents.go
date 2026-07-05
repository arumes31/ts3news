package bot

// Generic Abyss talents: the Deep-Delver extension and the per-spec sub-trees.
// Levels are stored generically as key→level JSON in app_meta (no per-node DB
// column), and each allocated level's effect is folded into treeBonusFor so it
// rides the same live combat/economy pipeline as the skill web. See
// internal/content/abyss_talents.go for the node definitions.

import (
	"encoding/json"
	"fmt"
	"net/http"

	"ts3news/internal/content"
)

func abyssTalentKey(uid string) string { return "abyss_talents_" + uid }

// loadAbyssTalentLevels returns the player's allocated generic-talent levels.
func (b *Bot) loadAbyssTalentLevels(uid string) map[string]int {
	out := map[string]int{}
	var js string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssTalentKey(uid)).Scan(&js)
	if js != "" {
		_ = json.Unmarshal([]byte(js), &out)
	}
	return out
}

// saveAbyssTalentLevels persists the whole level map (upsert).
func (b *Bot) saveAbyssTalentLevels(uid string, m map[string]int) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssTalentKey(uid), string(data))
	return err
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
	cost := int64(level+1) * 10
	// Guarded debit: only proceeds if the player still has the tokens (matches the
	// legacy Deep-Delver spend). RowsAffected==0 means someone else spent first.
	res, err := s.bot.DB.Exec(
		"UPDATE users SET abyss_tokens = abyss_tokens - $1 WHERE client_uid=$2 AND abyss_tokens >= $1", cost, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough tokens"})
		return
	}
	levels[t.Key] = level + 1
	if err := s.bot.saveAbyssTalentLevels(uid, levels); err != nil {
		// Refund on persistence failure so tokens are never lost silently.
		_, _ = s.bot.DB.Exec("UPDATE users SET abyss_tokens = abyss_tokens + $1 WHERE client_uid=$2", cost, uid)
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
			refund += int64(l) * 10
		}
	}
	return refund
}
