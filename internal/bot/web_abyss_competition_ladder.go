package bot

import (
	"net/http"
	"time"
)

func (s *WebServer) handleAbyssShameOptIn(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		writeJSON(w, map[string]any{"ok": false, "error": "method"})
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid request"})
		return
	}
	_, err := s.bot.DB.Exec(`INSERT INTO abyss_social_profiles (client_uid,shame_opt_in)
		VALUES ($1,$2) ON CONFLICT (client_uid) DO UPDATE SET shame_opt_in=EXCLUDED.shame_opt_in`, uid, req.Enabled)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "enabled": req.Enabled})
}

func (s *WebServer) handleAbyssWagerJoin(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		writeJSON(w, map[string]any{"ok": false, "error": "method"})
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}
	var req struct {
		Bracket int `json:"bracket"`
	}
	if readJSON(r, &req) != nil || (req.Bracket != 100 && req.Bracket != 1000 && req.Bracket != 10000) {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid bracket"})
		return
	}
	key, _, _ := abyssCompetitionWeekAt(time.Now())
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var gold int64
	if err := tx.QueryRow("SELECT gold FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&gold); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if gold < int64(req.Bracket) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
		return
	}
	result, err := tx.Exec(`INSERT INTO abyss_wager_entries (week_key,bracket,client_uid,entry_fee)
		VALUES ($1,$2,$3,$2) ON CONFLICT DO NOTHING`, key, req.Bracket, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	inserted, err := result.RowsAffected()
	if err != nil || inserted != 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "already joined"})
		return
	}
	if err := tx.QueryRow("UPDATE users SET gold=gold-$1 WHERE client_uid=$2 RETURNING gold", req.Bracket, uid).Scan(&gold); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "week": key, "bracket": req.Bracket, "gold": gold})
}
