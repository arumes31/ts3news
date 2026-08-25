package bot

import "net/http"

const abyssRunFlagFocus = "focus"

var abyssFocusIDs = map[string]int64{
	"balanced":  1,
	"gold":      2,
	"loot":      3,
	"xp":        4,
	"materials": 5,
	"tokens":    6,
}

var abyssFocusByID = map[int64]string{
	1: "balanced",
	2: "gold",
	3: "loot",
	4: "xp",
	5: "materials",
	6: "tokens",
}

func abyssFocusPreference(flags map[string]int64) string {
	return abyssFocusByID[flags[abyssRunFlagFocus]]
}

func (s *WebServer) selectedAbyssFocus(uid string, run abyssRun) string {
	if focus := abyssFocusPreference(s.bot.loadRunFlags(uid)); focus != "" {
		return focus
	}
	return s.autoSelectFocus(uid, run)
}

func (s *WebServer) handleAbyssFocus(w http.ResponseWriter, r *http.Request, uid string) {
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
		Focus string `json:"focus"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	run := s.bot.loadAbyssRun(uid)
	if !run.Active || run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "focus can only change during an active run"})
		return
	}
	focusID := int64(0)
	if req.Focus != "" && req.Focus != "auto" {
		var ok bool
		focusID, ok = abyssFocusIDs[req.Focus]
		if !ok {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid focus"})
			return
		}
	}
	if err := s.bot.setRunFlag(uid, abyssRunFlagFocus, focusID); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{
		"ok":         true,
		"preference": abyssFocusPreference(s.bot.loadRunFlags(uid)),
		"focus":      s.selectedAbyssFocus(uid, run),
	})
}
