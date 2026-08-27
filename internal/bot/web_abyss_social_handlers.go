package bot

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (s *WebServer) handleAbyssPetTrain(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var req struct {
		PetID int64 `json:"pet_id"`
	}
	if readJSON(r, &req) != nil || req.PetID <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid pet"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var strength, defense, speed, count int
	var rawProfile string
	if tx.QueryRow(`SELECT str,def,spd,CASE WHEN trained_on=CURRENT_DATE THEN training_count ELSE 0 END,autoskills::text FROM user_pets
		WHERE pet_id=$1 AND client_uid=$2 FOR UPDATE`, req.PetID, uid).Scan(&strength, &defense, &speed, &count, &rawProfile) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "pet not found"})
		return
	}
	if decodeAbyssPetProfile(rawProfile).busy(timeNowUTC()) {
		writeJSON(w, map[string]any{"ok": false, "error": "companion is committed to a stable activity"})
		return
	}
	if count >= abyssPetTrainingCap {
		writeJSON(w, map[string]any{"ok": false, "error": "daily pet training cap reached"})
		return
	}
	result, err := tx.Exec("UPDATE users SET gold=gold-$1 WHERE client_uid=$2 AND gold>=$1", abyssPetTrainingCost, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
		return
	}
	stat, value := abyssPetLowestStat(strength, defense, speed)
	gain := max(1, (value+99)/100)
	query := map[string]string{
		"str": "UPDATE user_pets SET str=str+$1,trained_on=CURRENT_DATE,training_count=$2 WHERE pet_id=$3 AND client_uid=$4",
		"def": "UPDATE user_pets SET def=def+$1,trained_on=CURRENT_DATE,training_count=$2 WHERE pet_id=$3 AND client_uid=$4",
		"spd": "UPDATE user_pets SET spd=spd+$1,trained_on=CURRENT_DATE,training_count=$2 WHERE pet_id=$3 AND client_uid=$4",
	}[stat]
	if _, err := tx.Exec(query, gain, count+1, req.PetID, uid); err != nil || tx.Commit() != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "stat": strings.ToUpper(stat), "gain": gain,
		"gold": s.bot.abyssGold(uid), "msg": fmt.Sprintf("Training complete: +%d %s.", gain, strings.ToUpper(stat))})
}

func (s *WebServer) handleAbyssPetSlot(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var req struct {
		PetID int64 `json:"pet_id"`
		Slot  int   `json:"slot"`
	}
	if readJSON(r, &req) != nil || req.PetID <= 0 || req.Slot < 0 || req.Slot > 2 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid active slot"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var prestige int
	if tx.QueryRow("SELECT abyss_prestige FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&prestige) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if req.Slot == 2 && prestige < 2 {
		writeJSON(w, map[string]any{"ok": false, "error": "second companion slot unlocks at Abyss prestige 2"})
		return
	}
	var rawProfile string
	if tx.QueryRow("SELECT autoskills::text FROM user_pets WHERE pet_id=$1 AND client_uid=$2 FOR UPDATE", req.PetID, uid).Scan(&rawProfile) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "pet not found"})
		return
	}
	if decodeAbyssPetProfile(rawProfile).busy(timeNowUTC()) {
		writeJSON(w, map[string]any{"ok": false, "error": "companion is committed to a stable activity"})
		return
	}
	if req.Slot > 0 {
		if _, err := tx.Exec("UPDATE user_pets SET active_slot=0 WHERE client_uid=$1 AND active_slot=$2", uid, req.Slot); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if _, err := tx.Exec("UPDATE user_pets SET active_slot=$1 WHERE pet_id=$2 AND client_uid=$3", req.Slot, req.PetID, uid); err != nil || tx.Commit() != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": "Companion formation updated."})
}

func (s *WebServer) handleAbyssPetAutoskill(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var req struct {
		PetID   int64  `json:"pet_id"`
		Ability string `json:"ability"`
		Enabled bool   `json:"enabled"`
	}
	if readJSON(r, &req) != nil || req.PetID <= 0 || req.Ability != "heal" {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid pet ability"})
		return
	}
	result, err := s.bot.DB.Exec(`UPDATE user_pets SET autoskills=jsonb_set(autoskills,'{heal}',to_jsonb($1::boolean),TRUE)
		WHERE pet_id=$2 AND client_uid=$3`, req.Enabled, req.PetID, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "pet not found"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "enabled": req.Enabled, "msg": "Companion autoskill updated."})
}

func (s *WebServer) handleAbyssBankFeedToggle(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	_, err := s.bot.DB.Exec(`INSERT INTO abyss_social_profiles (client_uid,bank_feed_opt_in) VALUES ($1,$2)
		ON CONFLICT (client_uid) DO UPDATE SET bank_feed_opt_in=EXCLUDED.bank_feed_opt_in`, uid, req.Enabled)
	writeJSON(w, map[string]any{"ok": err == nil, "enabled": req.Enabled, "msg": "Bank feed preference saved."})
}

func (s *WebServer) handleAbyssRivalClaim(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	week := abyssCurrentWeek(time.Now())
	var target, current int
	var claimed sql.NullTime
	if tx.QueryRow(`SELECT r.target_depth,r.claimed_at,u.abyss_best_depth FROM abyss_weekly_rivals r
		JOIN users u ON u.client_uid=r.client_uid WHERE r.week_key=$1 AND r.client_uid=$2 FOR UPDATE`, week, uid).
		Scan(&target, &claimed, &current) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "weekly rival unavailable"})
		return
	}
	if claimed.Valid {
		writeJSON(w, map[string]any{"ok": false, "error": "weekly rival reward already claimed"})
		return
	}
	if current <= target {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("reach depth %d to pass your rival", target+1)})
		return
	}
	if _, err := tx.Exec("UPDATE abyss_weekly_rivals SET claimed_at=NOW() WHERE week_key=$1 AND client_uid=$2", week, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec("UPDATE users SET abyss_tokens=abyss_tokens+$1 WHERE client_uid=$2", abyssRivalReward, uid); err != nil || tx.Commit() != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "tokens": s.bot.abyssTokens(uid), "msg": fmt.Sprintf("Rival passed: +%d Abyss Tokens.", abyssRivalReward)})
}

func (s *WebServer) handleAbyssWeeklyBossStrike(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	damage := abyssWeeklyBossDamage(s.bot.abyssPlayerCR(uid))
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	now := time.Now()
	week, name := abyssWeeklyBossDefinition(now)
	if _, err := tx.Exec(`INSERT INTO abyss_weekly_bosses (week_key,boss_name,max_hp,current_hp)
		VALUES ($1,$2,$3,$3) ON CONFLICT (week_key) DO NOTHING`, week, name, abyssWeeklyBossHP); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	var hp, maxHP int64
	var defeated sql.NullTime
	if tx.QueryRow("SELECT boss_name,current_hp,max_hp,defeated_at FROM abyss_weekly_bosses WHERE week_key=$1 FOR UPDATE", week).
		Scan(&name, &hp, &maxHP, &defeated) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if defeated.Valid || hp <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "the weekly server boss is already defeated"})
		return
	}
	drop := abyssWeeklyBossDropFor(name, week, uid, now)
	damage, drop = applyAbyssWorldBossWeekendReward(now, damage, drop)
	damage = min(damage, hp)
	loot := abyssWeeklyBossDropLabel(drop)
	result, err := tx.Exec(`INSERT INTO abyss_weekly_boss_contributions
		(week_key,client_uid,damage,loot_label) VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`, week, uid, damage, loot)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "you already contributed today"})
		return
	}
	if err := grantMaterialQ(tx, uid, drop.Material, drop.Amount); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	newHP := max(int64(0), hp-damage)
	if _, err := tx.Exec(`UPDATE abyss_weekly_bosses SET current_hp=$1,
		defeated_at=CASE WHEN $1=0 THEN NOW() ELSE defeated_at END WHERE week_key=$2`, newHP, week); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defeatedNow := hp > 0 && newHP == 0
	if defeatedNow {
		if _, err := tx.Exec(`UPDATE users u SET abyss_tokens=abyss_tokens+25
			FROM (SELECT DISTINCT client_uid FROM abyss_weekly_boss_contributions WHERE week_key=$1) c
			WHERE u.client_uid=c.client_uid`, week); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "damage": damage, "hp": newHP, "max_hp": maxHP, "loot": loot,
		"defeated": defeatedNow, "tokens": s.bot.abyssTokens(uid), "msg": fmt.Sprintf("Server boss contribution: %d damage · %s.", damage, loot)})
}

func abyssWeeklyBossDamage(combatRating float64) int64 {
	return max(int64(1), int64(combatRating*25))
}
