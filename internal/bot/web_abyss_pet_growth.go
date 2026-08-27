package bot

import (
	"fmt"
	"net/http"
)

type abyssPetGrowthRow struct {
	id         int64
	name       string
	mobType    string
	level      int
	hp         int
	maxHP      int
	strength   int
	defense    int
	speed      int
	loyalty    int
	activeSlot int
	rawProfile string
}

func (s *WebServer) handleAbyssPetFusion(w http.ResponseWriter, r *http.Request, uid string) {
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
		KeepID  int64 `json:"keep_pet_id"`
		DonorID int64 `json:"donor_pet_id"`
	}
	if readJSON(r, &req) != nil || req.KeepID <= 0 || req.DonorID <= 0 || req.KeepID == req.DonorID {
		writeJSON(w, map[string]any{"ok": false, "error": "choose two different companions"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.Query(`SELECT pet_id,name,mob_type,level,hp,max_hp,str,def,spd,loyalty,active_slot,autoskills::text
		FROM user_pets WHERE client_uid=$1 AND pet_id IN ($2,$3) ORDER BY pet_id FOR UPDATE`, uid, req.KeepID, req.DonorID)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	pets := make(map[int64]abyssPetGrowthRow, 2)
	for rows.Next() {
		var pet abyssPetGrowthRow
		if err := rows.Scan(&pet.id, &pet.name, &pet.mobType, &pet.level, &pet.hp, &pet.maxHP,
			&pet.strength, &pet.defense, &pet.speed, &pet.loyalty, &pet.activeSlot, &pet.rawProfile); err != nil {
			_ = rows.Close()
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		pets[pet.id] = pet
	}
	if err := rows.Close(); err != nil || rows.Err() != nil || len(pets) != 2 {
		writeJSON(w, map[string]any{"ok": false, "error": "companions not found"})
		return
	}
	keep, donor := pets[req.KeepID], pets[req.DonorID]
	donorProfile := decodeAbyssPetProfile(donor.rawProfile)
	if keep.mobType != donor.mobType {
		writeJSON(w, map[string]any{"ok": false, "error": "fusion requires two companions from the same family"})
		return
	}
	if donor.activeSlot != 0 || donorProfile.Favorite || donorProfile.busy(timeNowUTC()) {
		writeJSON(w, map[string]any{"ok": false, "error": "the fusion donor must be an unfavorited reserve companion"})
		return
	}
	keepProfile := decodeAbyssPetProfile(keep.rawProfile)
	keepProfile.FusionRank++
	encoded, err := encodeAbyssPetProfile(keepProfile)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "profile"})
		return
	}
	newMaxHP := keep.maxHP + abyssPetFusionGain(keep.maxHP, donor.maxHP)
	newSTR := keep.strength + abyssPetFusionGain(keep.strength, donor.strength)
	newDEF := keep.defense + abyssPetFusionGain(keep.defense, donor.defense)
	newSPD := keep.speed + abyssPetFusionGain(keep.speed, donor.speed)
	if _, err := tx.Exec("DELETE FROM user_pets WHERE pet_id=$1 AND client_uid=$2", donor.id, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec(`UPDATE user_pets SET hp=$1,max_hp=$1,str=$2,def=$3,spd=$4,
		loyalty=LEAST(100,loyalty+5),autoskills=$5::jsonb WHERE pet_id=$6 AND client_uid=$7`,
		newMaxHP, newSTR, newDEF, newSPD, encoded, keep.id, uid); err != nil || tx.Commit() != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "rank": keepProfile.FusionRank,
		"msg": fmt.Sprintf("%s absorbed %s and reached fusion rank %d.", keep.name, donor.name, keepProfile.FusionRank)})
}

func (s *WebServer) handleAbyssPetRevive(w http.ResponseWriter, r *http.Request, uid string) {
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
		MemorialID int64 `json:"memorial_id"`
	}
	if readJSON(r, &req) != nil || req.MemorialID <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid memorial"})
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
	var name, mobType string
	var level, loyalty int
	if err := tx.QueryRow(`SELECT name,mob_type,level,loyalty FROM abyss_pet_memorials
		WHERE id=$1 AND client_uid=$2 FOR UPDATE`, req.MemorialID, uid).Scan(&name, &mobType, &level, &loyalty); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "memorial not found"})
		return
	}
	limit, err := abyssPetStableLimitTx(tx, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	var owned int
	if err := tx.QueryRow("SELECT COUNT(*) FROM user_pets WHERE client_uid=$1", uid).Scan(&owned); err != nil || owned >= limit {
		writeJSON(w, map[string]any{"ok": false, "error": "make room in the stable before reviving a companion"})
		return
	}
	var feathers int
	if err := tx.QueryRow(`SELECT remaining_fights FROM user_consumables
		WHERE client_uid=$1 AND cons_id='pet_revival_scroll' FOR UPDATE`, uid).Scan(&feathers); err != nil || feathers <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "a Companion Revival Scroll is required"})
		return
	}
	if feathers == 1 {
		_, err = tx.Exec("DELETE FROM user_consumables WHERE client_uid=$1 AND cons_id='pet_revival_scroll'", uid)
	} else {
		_, err = tx.Exec(`UPDATE user_consumables SET remaining_fights=remaining_fights-1
			WHERE client_uid=$1 AND cons_id='pet_revival_scroll'`, uid)
	}
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	hp, strength, defense, speed := abyssPetRevivedStats(level)
	profile, err := encodeAbyssPetProfile(abyssPetProfile{BarkStyle: "gentle"})
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "profile"})
		return
	}
	if _, err := tx.Exec(`INSERT INTO user_pets
		(client_uid,name,mob_type,level,hp,max_hp,str,def,spd,loyalty,active_slot,autoskills)
		VALUES ($1,$2,$3,$4,$5,$5,$6,$7,$8,$9,0,$10::jsonb)`, uid, name, mobType,
		level, hp, strength, defense, speed, max(50, loyalty), profile); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec("DELETE FROM abyss_pet_memorials WHERE id=$1 AND client_uid=$2", req.MemorialID, uid); err != nil || tx.Commit() != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": name + " rises from the memorial flame."})
}
