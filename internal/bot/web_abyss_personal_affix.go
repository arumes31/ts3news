package bot

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"
)

const (
	abyssPersonalAffixRerollCost = 10
	abyssRunFlagDailyAffix       = "daily_affix"
)

type abyssPersonalAffixView struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Rerolled  bool   `json:"rerolled"`
	Available bool   `json:"available"`
	Cost      int    `json:"cost"`
}

func abyssPersonalAffixKey(uid string, at time.Time) string {
	return "abyss_personal_affix_" + uid + "_" + at.UTC().Format("2006-01-02")
}

func abyssDailyAffixIndex(key string) int64 {
	for index, candidate := range abyssDailyMods {
		if candidate == key {
			return int64(index + 1)
		}
	}
	return 0
}

func chooseAbyssAffixReroll(source io.Reader, current string) (string, error) {
	eligible := make([]string, 0, len(abyssDailyMods)-1)
	for _, affix := range abyssDailyMods {
		if affix != current {
			eligible = append(eligible, affix)
		}
	}
	if len(eligible) == 0 {
		return "", fmt.Errorf("no alternate affix available")
	}
	draw, err := rand.Int(source, big.NewInt(int64(len(eligible))))
	if err != nil {
		return "", fmt.Errorf("reroll personal affix: %w", err)
	}
	return eligible[draw.Int64()], nil
}

func (b *Bot) personalAbyssAffixAt(uid string, at time.Time) (string, bool) {
	var affix string
	err := b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssPersonalAffixKey(uid, at)).Scan(&affix)
	if err != nil || abyssDailyAffixIndex(affix) == 0 {
		return "", false
	}
	return affix, true
}

func (b *Bot) currentPersonalAbyssAffixAt(uid string, at time.Time) (int64, string, bool) {
	seed, global := b.currentDailyChallengeAt(at)
	if personal, ok := b.personalAbyssAffixAt(uid, at); ok {
		return seed, personal, true
	}
	return seed, global, false
}

func (b *Bot) abyssRunDailyChallenge(uid string) (int64, string) {
	flags := b.loadRunFlags(uid)
	if flags[abyssRunFlagDailyAffix] == -1 {
		seed, _ := b.currentDailyChallenge()
		return seed, ""
	}
	index := int(flags[abyssRunFlagDailyAffix]) - 1
	if index >= 0 && index < len(abyssDailyMods) {
		seed, _ := b.currentDailyChallenge()
		return seed, abyssDailyMods[index]
	}
	seed, affix, _ := b.currentPersonalAbyssAffixAt(uid, time.Now().UTC())
	return seed, affix
}

func (b *Bot) abyssPersonalAffixView(uid string, at time.Time) abyssPersonalAffixView {
	_, affix, rerolled := b.currentPersonalAbyssAffixAt(uid, at)
	return abyssPersonalAffixView{
		Key: affix, Label: abyssDailyAffixLabel(affix), Rerolled: rerolled,
		Available: !rerolled, Cost: abyssPersonalAffixRerollCost,
	}
}

func (s *WebServer) handleAbyssPersonalAffixReroll(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.bot.loadAbyssRun(uid).Active {
		writeJSON(w, map[string]any{"ok": false, "error": "finish the active run before rerolling tomorrow's run affix"})
		return
	}
	at := time.Now().UTC()
	if _, exists := s.bot.personalAbyssAffixAt(uid, at); exists {
		writeJSON(w, map[string]any{"ok": false, "error": "personal affix already rerolled today"})
		return
	}
	_, current := s.bot.currentDailyChallengeAt(at)
	selected, err := chooseAbyssAffixReroll(rand.Reader, current)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "affix reroll unavailable"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.Exec(
		"INSERT INTO app_meta (key, value) VALUES ($1,$2) ON CONFLICT DO NOTHING",
		abyssPersonalAffixKey(uid, at), selected,
	)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if inserted, _ := res.RowsAffected(); inserted == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "personal affix already rerolled today"})
		return
	}
	res, err = tx.Exec(
		"UPDATE users SET abyss_tokens=abyss_tokens-$1 WHERE client_uid=$2 AND abyss_tokens >= $1",
		abyssPersonalAffixRerollCost, uid,
	)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if charged, _ := res.RowsAffected(); charged == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Abyss Tokens"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	view := s.bot.abyssPersonalAffixView(uid, at)
	writeJSON(w, map[string]any{"ok": true, "affix": view, "tokens": s.bot.abyssTokens(uid)})
}
