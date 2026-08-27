package bot

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (s *WebServer) handleAbyssPetActivity(w http.ResponseWriter, r *http.Request, uid string) {
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
		Action string `json:"action"`
		Kind   string `json:"kind"`
	}
	if readJSON(r, &req) != nil || req.PetID <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid companion activity"})
		return
	}
	req.Action = strings.TrimSpace(req.Action)
	req.Kind = strings.TrimSpace(req.Kind)
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var name, rawProfile string
	var level, hp, maxHP, activeSlot int
	if err := tx.QueryRow(`SELECT name,level,hp,max_hp,active_slot,autoskills::text FROM user_pets
		WHERE pet_id=$1 AND client_uid=$2 FOR UPDATE`, req.PetID, uid).
		Scan(&name, &level, &hp, &maxHP, &activeSlot, &rawProfile); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "companion not found"})
		return
	}
	profile := decodeAbyssPetProfile(rawProfile)
	now := timeNowUTC()
	message := ""
	switch req.Action {
	case "daycare_start":
		if activeSlot != 0 || profile.busy(now) {
			writeJSON(w, map[string]any{"ok": false, "error": "only an idle reserve companion can enter daycare"})
			return
		}
		profile.DaycareSince = now.Format(time.RFC3339)
		message = name + " entered daycare and will gain 5 bond XP per hour."
	case "daycare_claim":
		xp := abyssPetDaycareXP(profile, now)
		if xp <= 0 {
			writeJSON(w, map[string]any{"ok": false, "error": "daycare has no bond XP ready"})
			return
		}
		profile.DaycareSince = ""
		profile.XP += xp
		for profile.XP >= abyssPetXPThreshold(level) {
			profile.XP -= abyssPetXPThreshold(level)
			level++
			maxHP += max(5, maxHP/20)
		}
		hp = maxHP
		message = fmt.Sprintf("%s returned from daycare with %d bond XP.", name, xp)
	case "expedition_start":
		if activeSlot != 0 || profile.busy(now) {
			writeJSON(w, map[string]any{"ok": false, "error": "only an idle reserve companion can start an expedition"})
			return
		}
		if req.Kind != "dust" && req.Kind != "crystal" && req.Kind != "prism" {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid expedition"})
			return
		}
		profile.ExpeditionKind = req.Kind
		profile.ExpeditionUntil = now.Add(abyssPetExpeditionDuration).Format(time.RFC3339)
		message = fmt.Sprintf("%s began an eight-hour %s expedition.", name, req.Kind)
	case "expedition_claim":
		if profile.ExpeditionKind == "" || !abyssPetExpeditionReady(profile, now) {
			writeJSON(w, map[string]any{"ok": false, "error": "expedition reward is not ready"})
			return
		}
		material, amount := abyssPetExpeditionReward(profile.ExpeditionKind, level)
		if err := grantMaterialQ(tx, uid, material, amount); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		profile.ExpeditionKind = ""
		profile.ExpeditionUntil = ""
		message = fmt.Sprintf("%s returned with %d %s.", name, amount, material)
	default:
		writeJSON(w, map[string]any{"ok": false, "error": "invalid companion activity"})
		return
	}
	encoded, err := encodeAbyssPetProfile(profile)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "profile"})
		return
	}
	if _, err := tx.Exec(`UPDATE user_pets SET level=$1,hp=$2,max_hp=$3,autoskills=$4::jsonb
		WHERE pet_id=$5 AND client_uid=$6`, level, hp, maxHP, encoded, req.PetID, uid); err != nil || tx.Commit() != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": message})
}
