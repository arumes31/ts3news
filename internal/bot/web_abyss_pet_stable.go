package bot

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// handleAbyssPetManage owns the low-frequency stable mutations. Keeping them
// behind one transaction prevents the profile JSON and the pet row from
// drifting apart when a request fails midway through an update.
func (s *WebServer) handleAbyssPetManage(w http.ResponseWriter, r *http.Request, uid string) {
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
		PetID           int64  `json:"pet_id"`
		Action          string `json:"action"`
		Name            string `json:"name"`
		Style           string `json:"style"`
		ConfirmFavorite bool   `json:"confirm_favorite"`
	}
	if readJSON(r, &req) != nil || req.PetID <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid companion"})
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
	var message string
	switch req.Action {
	case "rename":
		newName := strings.TrimSpace(req.Name)
		if !abyssPetNameValid(newName) {
			writeJSON(w, map[string]any{"ok": false, "error": "use 1–24 safe letters, numbers, spaces, apostrophes or hyphens"})
			return
		}
		if _, err := tx.Exec("UPDATE user_pets SET name=$1 WHERE pet_id=$2 AND client_uid=$3", newName, req.PetID, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		name = newName
		message = "Companion renamed to " + newName + "."
	case "favorite":
		profile.Favorite = !profile.Favorite
		encoded, err := encodeAbyssPetProfile(profile)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "profile"})
			return
		}
		if _, err := tx.Exec("UPDATE user_pets SET autoskills=$1::jsonb WHERE pet_id=$2 AND client_uid=$3", encoded, req.PetID, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if profile.Favorite {
			message = name + " marked as a favorite."
		} else {
			message = name + " removed from favorites."
		}
	case "bark":
		if req.Style != "quiet" && req.Style != "gentle" && req.Style != "bold" {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid companion voice"})
			return
		}
		profile.BarkStyle = req.Style
		encoded, err := encodeAbyssPetProfile(profile)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "profile"})
			return
		}
		if _, err := tx.Exec("UPDATE user_pets SET autoskills=$1::jsonb WHERE pet_id=$2 AND client_uid=$3", encoded, req.PetID, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		message = name + " voice set to " + req.Style + "."
	case "release":
		if profile.busy(timeNowUTC()) {
			writeJSON(w, map[string]any{"ok": false, "error": "companion is committed to a stable activity"})
			return
		}
		if profile.Favorite && !req.ConfirmFavorite {
			writeJSON(w, map[string]any{"ok": false, "error": "favorite confirmation required", "confirm_required": true})
			return
		}
		if _, err := tx.Exec("DELETE FROM user_pets WHERE pet_id=$1 AND client_uid=$2", req.PetID, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		message = name + " was released from your stable."
	default:
		writeJSON(w, map[string]any{"ok": false, "error": "invalid stable action"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "name": name, "favorite": profile.Favorite, "msg": message})
}

func (s *WebServer) handleAbyssPetFeed(w http.ResponseWriter, r *http.Request, uid string) {
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
		PetID  int64  `json:"pet_id"`
		ConsID string `json:"cons_id"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid companion"})
		return
	}
	req.ConsID = strings.TrimSpace(req.ConsID)
	if req.PetID <= 0 || req.ConsID == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid companion"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var owner string
	if err := tx.QueryRow("SELECT client_uid FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&owner); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	var name, rawProfile string
	var level, hp, maxHP, loyalty int
	if err := tx.QueryRow(`SELECT name,level,hp,max_hp,loyalty,autoskills::text FROM user_pets
		WHERE pet_id=$1 AND client_uid=$2 FOR UPDATE`, req.PetID, uid).
		Scan(&name, &level, &hp, &maxHP, &loyalty, &rawProfile); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "companion not found"})
		return
	}
	profile := decodeAbyssPetProfile(rawProfile)
	if profile.busy(timeNowUTC()) {
		writeJSON(w, map[string]any{"ok": false, "error": "companion is away from the stable"})
		return
	}
	var charges int
	if err := tx.QueryRow(`SELECT remaining_fights FROM user_consumables
		WHERE client_uid=$1 AND cons_id=$2 FOR UPDATE`, uid, req.ConsID).Scan(&charges); err != nil || charges <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "selected consumable is unavailable"})
		return
	}
	if charges == 1 {
		_, err = tx.Exec("DELETE FROM user_consumables WHERE client_uid=$1 AND cons_id=$2", uid, req.ConsID)
	} else {
		_, err = tx.Exec(`UPDATE user_consumables SET remaining_fights=remaining_fights-1
			WHERE client_uid=$1 AND cons_id=$2`, uid, req.ConsID)
	}
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	profile.XP += max(25, level*10)
	levelled := false
	for profile.XP >= abyssPetXPThreshold(level) {
		profile.XP -= abyssPetXPThreshold(level)
		level++
		maxHP += max(5, maxHP/20)
		levelled = true
	}
	hp = maxHP
	loyalty = min(100, loyalty+10)
	encoded, err := encodeAbyssPetProfile(profile)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "profile"})
		return
	}
	if _, err := tx.Exec(`UPDATE user_pets SET level=$1,hp=$2,max_hp=$2,loyalty=$3,autoskills=$4::jsonb
		WHERE pet_id=$5 AND client_uid=$6`, level, maxHP, loyalty, encoded, req.PetID, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	message := fmt.Sprintf("%s is fed, fully healed, and gained loyalty.", name)
	if levelled {
		message = fmt.Sprintf("%s reached level %d!", name, level)
	}
	writeJSON(w, map[string]any{"ok": true, "level": level,
		"hp": maxHP, "max_hp": maxHP, "loyalty": loyalty, "xp": profile.XP,
		"xp_next": abyssPetXPThreshold(level), "consumables": s.bot.getConsumables(uid), "msg": message})
}

func timeNowUTC() time.Time {
	return time.Now().UTC()
}
