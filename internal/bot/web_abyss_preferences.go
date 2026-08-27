package bot

import (
	"net/http"
	"strings"
)

const abyssFontSizeDefault = "m"

func abyssFontSizeKey(uid string) string { return "abyss_font_size_" + uid }

func normalizeAbyssFontSize(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "s", "m", "l":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return abyssFontSizeDefault
	}
}

func (b *Bot) loadAbyssFontSize(uid string) string {
	var value string
	if b == nil || b.DB == nil || b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssFontSizeKey(uid)).Scan(&value) != nil {
		return abyssFontSizeDefault
	}
	return normalizeAbyssFontSize(value)
}

func (b *Bot) saveAbyssFontSize(uid, value string) error {
	_, err := b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, abyssFontSizeKey(uid), normalizeAbyssFontSize(value))
	return err
}

func (s *WebServer) handleAbyssFontSize(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method == http.MethodGet {
		writeJSON(w, map[string]any{"ok": true, "font_size": s.bot.loadAbyssFontSize(uid)})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		FontSize string `json:"font_size"`
	}
	if readJSON(r, &req) != nil || normalizeAbyssFontSize(req.FontSize) != strings.ToLower(strings.TrimSpace(req.FontSize)) {
		writeJSON(w, map[string]any{"ok": false, "error": "font size must be s, m, or l"})
		return
	}
	if err := s.bot.saveAbyssFontSize(uid, req.FontSize); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "could not save font size"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "font_size": normalizeAbyssFontSize(req.FontSize)})
}
