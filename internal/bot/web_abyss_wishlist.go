package bot

import (
	"log"
	"net/http"
)

func (s *WebServer) handleAbyssWishlist(w http.ResponseWriter, r *http.Request, uid string) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, map[string]any{
			"ok":       true,
			"wishlist": abyssWishlistViewFor(s.bot.loadAbyssWishlist(uid), r.URL.Query().Get("q")),
		})
	case http.MethodPost:
		unlock := s.lockAbyss(uid)
		defer unlock()
		if s.rejectDuringLiveCombat(w, uid) {
			return
		}
		var request struct {
			GearID string `json:"gear_id"`
		}
		if readJSON(r, &request) != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
			return
		}
		state, err := toggleAbyssWishlist(s.bot.loadAbyssWishlist(uid), request.GearID)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		if err := saveAbyssWishlist(s.bot.DB, uid, state); err != nil {
			log.Printf("abyss wishlist update failed for %s: %v", uid, err)
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		writeJSON(w, map[string]any{
			"ok":       true,
			"wishlist": abyssWishlistViewFor(state, r.URL.Query().Get("q")),
		})
	default:
		http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
	}
}
