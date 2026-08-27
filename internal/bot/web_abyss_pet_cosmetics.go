package bot

import (
	"net/http"
	"slices"
	"strings"
)

func (s *WebServer) handleAbyssPetCosmetic(w http.ResponseWriter, r *http.Request, uid string) {
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
		PetID int64  `json:"pet_id"`
		Key   string `json:"key"`
	}
	if readJSON(r, &req) != nil || req.PetID <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid companion cosmetic"})
		return
	}
	req.Key = strings.TrimSpace(req.Key)
	cosmetic, known := abyssPetCosmeticByKey(req.Key)
	if req.Key != "" && !known {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown companion cosmetic"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var name, rawProfile string
	if err := tx.QueryRow(`SELECT name,autoskills::text FROM user_pets
		WHERE pet_id=$1 AND client_uid=$2 FOR UPDATE`, req.PetID, uid).Scan(&name, &rawProfile); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "companion not found"})
		return
	}
	profile := decodeAbyssPetProfile(rawProfile)
	charged := 0
	if req.Key != "" && !slices.Contains(profile.OwnedCosmetics, req.Key) {
		result, err := tx.Exec("UPDATE users SET abyss_tokens=abyss_tokens-$1 WHERE client_uid=$2 AND abyss_tokens>=$1", abyssPetCosmeticCost, uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if changed, _ := result.RowsAffected(); changed != 1 {
			writeJSON(w, map[string]any{"ok": false, "error": "not enough Abyss Tokens"})
			return
		}
		profile.OwnedCosmetics = append(profile.OwnedCosmetics, req.Key)
		charged = abyssPetCosmeticCost
	}
	profile.Cosmetic = req.Key
	encoded, err := encodeAbyssPetProfile(profile)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "profile"})
		return
	}
	if _, err := tx.Exec("UPDATE user_pets SET autoskills=$1::jsonb WHERE pet_id=$2 AND client_uid=$3", encoded, req.PetID, uid); err != nil || tx.Commit() != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	message := name + " returned to its natural appearance."
	if req.Key != "" {
		message = cosmetic.Icon + " " + cosmetic.Name + " equipped on " + name + ". Cosmetic only."
	}
	writeJSON(w, map[string]any{"ok": true, "charged": charged, "tokens": s.bot.abyssTokens(uid), "msg": message})
}
