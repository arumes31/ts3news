package bot

import (
	"encoding/json"
	"net/http"
)

func (s *WebServer) handleAbyssSkillPriority(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}

	var req struct {
		SkillPriority []string `json:"skill_priority"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	priority, err := validateAbyssSkillPriority(req.SkillPriority, s.bot.getSkills(uid))
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	encoded, err := json.Marshal(priority)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "skill priority could not be encoded"})
		return
	}
	if err := s.bot.setAbyssCombatOption(uid, "skill_priority", string(encoded)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "skill priority could not be saved"})
		return
	}
	writeJSON(w, map[string]any{
		"ok": true, "skill_priority": priority,
		"msg": "Automatic skill priority saved for the next combat.",
	})
}
