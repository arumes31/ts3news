package bot

import (
	"net/http"
	"strings"
	"time"
)

func (s *WebServer) handleAbyssPetGiftCreate(w http.ResponseWriter, r *http.Request, uid string) {
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
		PetID     int64  `json:"pet_id"`
		Recipient string `json:"recipient_uid"`
	}
	if readJSON(r, &req) != nil || req.PetID <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid companion gift"})
		return
	}
	req.Recipient = strings.TrimSpace(req.Recipient)
	if req.Recipient == "" || req.Recipient == uid {
		writeJSON(w, map[string]any{"ok": false, "error": "choose another player by exact user ID"})
		return
	}
	code, err := abyssPetGiftCode()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "could not create a secure gift code"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var recipient string
	if err := tx.QueryRow("SELECT client_uid FROM users WHERE client_uid=$1 FOR UPDATE", req.Recipient).Scan(&recipient); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "recipient not found"})
		return
	}
	var name, rawProfile string
	var activeSlot int
	if err := tx.QueryRow(`SELECT name,active_slot,autoskills::text FROM user_pets
		WHERE pet_id=$1 AND client_uid=$2 FOR UPDATE`, req.PetID, uid).Scan(&name, &activeSlot, &rawProfile); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "companion not found"})
		return
	}
	profile := decodeAbyssPetProfile(rawProfile)
	now := timeNowUTC()
	if activeSlot != 0 || profile.Favorite || profile.busy(now) {
		writeJSON(w, map[string]any{"ok": false, "error": "only an unfavorited idle reserve companion can be gifted"})
		return
	}
	if _, err := tx.Exec("DELETE FROM abyss_pet_gifts WHERE pet_id=$1 AND expires_at<=NOW()", req.PetID); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	profile.GiftUntil = now.Add(abyssPetGiftLifetime).Format(time.RFC3339)
	encoded, err := encodeAbyssPetProfile(profile)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "profile"})
		return
	}
	if _, err := tx.Exec("UPDATE user_pets SET autoskills=$1::jsonb WHERE pet_id=$2 AND client_uid=$3", encoded, req.PetID, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec(`INSERT INTO abyss_pet_gifts (code,pet_id,sender_uid,recipient_uid,expires_at)
		VALUES ($1,$2,$3,$4,$5)`, code, req.PetID, uid, recipient, timeNowUTC().Add(abyssPetGiftLifetime)); err != nil || tx.Commit() != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "companion already has a pending gift"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "code": code,
		"msg": name + " is reserved for the recipient for seven days."})
}

func (s *WebServer) handleAbyssPetGiftClaim(w http.ResponseWriter, r *http.Request, uid string) {
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
		Code string `json:"code"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid gift code"})
		return
	}
	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
	if req.Code == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid gift code"})
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
	var petID int64
	var sender, name, rawProfile string
	if err := tx.QueryRow(`SELECT g.pet_id,g.sender_uid,p.name,p.autoskills::text FROM abyss_pet_gifts g
		JOIN user_pets p ON p.pet_id=g.pet_id AND p.client_uid=g.sender_uid
		WHERE g.code=$1 AND g.recipient_uid=$2 AND g.expires_at>NOW() FOR UPDATE OF g,p`, req.Code, uid).
		Scan(&petID, &sender, &name, &rawProfile); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "gift code invalid, expired, or meant for another player"})
		return
	}
	limit, err := abyssPetStableLimitTx(tx, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	var owned int
	if err := tx.QueryRow("SELECT COUNT(*) FROM user_pets WHERE client_uid=$1", uid).Scan(&owned); err != nil || owned >= limit {
		writeJSON(w, map[string]any{"ok": false, "error": "make room in the stable before claiming this gift"})
		return
	}
	profile := decodeAbyssPetProfile(rawProfile)
	profile.GiftUntil = ""
	encoded, err := encodeAbyssPetProfile(profile)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "profile"})
		return
	}
	if _, err := tx.Exec(`UPDATE user_pets SET client_uid=$1,active_slot=0,autoskills=$2::jsonb
		WHERE pet_id=$3 AND client_uid=$4`, uid, encoded, petID, sender); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec("DELETE FROM abyss_pet_gifts WHERE code=$1", req.Code); err != nil || tx.Commit() != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": name + " joined your stable as a bound gift."})
}
