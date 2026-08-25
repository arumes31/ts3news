package bot

import "net/http"

func (s *WebServer) handleAbyssCombatSettings(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method == http.MethodGet {
		writeJSON(w, map[string]any{"ok": true, "hold_mana": s.bot.abyssHoldMana(uid)})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		HoldMana bool `json:"hold_mana"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	value := ""
	if req.HoldMana {
		value = "1"
	}
	if err := s.bot.setAbyssCombatOption(uid, "hold_mana", value); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "combat setting could not be saved"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "hold_mana": req.HoldMana})
}
