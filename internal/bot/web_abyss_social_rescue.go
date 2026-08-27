package bot

import (
	"net/http"
)

func (s *WebServer) handleAbyssRescueMission(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		DeathID int64 `json:"death_id"`
	}
	if readJSON(r, &req) != nil || req.DeathID <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "choose a friend's recoverable death"})
		return
	}
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var owner string
	var depth int
	var lostCache int64
	if err := tx.QueryRowContext(r.Context(), `SELECT d.client_uid,d.depth,d.lost_cache FROM abyss_deaths d
		WHERE d.id=$1 AND d.client_uid<>$2 AND d.rescued_at IS NULL AND d.lost_cache>0
		AND EXISTS(SELECT 1 FROM abyss_friendships f WHERE f.uid_low=LEAST($2,d.client_uid)
		AND f.uid_high=GREATEST($2,d.client_uid) AND f.accepted_at IS NOT NULL) FOR UPDATE`,
		req.DeathID, uid).Scan(&owner, &depth, &lostCache); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "recoverable friend death not found"})
		return
	}
	if _, err := tx.ExecContext(r.Context(), `INSERT INTO abyss_rescue_missions
		(death_id,rescuer_uid,owner_uid,depth,lost_cache) VALUES ($1,$2,$3,$4,$5)`,
		req.DeathID, uid, owner, depth, lostCache); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "finish your active rescue mission first"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": "Rescue accepted. Reach the fallen depth to recover 10% of the lost cache for both friends."})
}
