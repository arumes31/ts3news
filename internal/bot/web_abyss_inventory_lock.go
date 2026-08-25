package bot

import (
	"net/http"
	"time"
)

const abyssRecentlyLootedWindow = 10 * time.Minute

func abyssRecentlyLooted(acquiredAt, now time.Time) bool {
	age := now.Sub(acquiredAt)
	return age >= 0 && age <= abyssRecentlyLootedWindow
}

func (s *WebServer) handleAbyssInventoryLock(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var request struct {
		InvID  int64 `json:"inv_id"`
		Locked bool  `json:"locked"`
	}
	if readJSON(r, &request) != nil || request.InvID <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid inventory item"})
		return
	}
	result, err := s.bot.DB.Exec("UPDATE user_inventory SET locked=$1 WHERE id=$2 AND client_uid=$3", request.Locked, request.InvID, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "locked": request.Locked})
}
