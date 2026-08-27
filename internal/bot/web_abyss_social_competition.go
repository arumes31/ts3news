package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ts3news/internal/content"
)

func abyssDuelResolve(firstName string, first content.Stats, secondName string, second content.Stats) (int, []string) {
	firstHP := max(1, first.HP)
	secondHP := max(1, second.HP)
	firstAttack := max(1, first.STR+first.INT/2)
	secondAttack := max(1, second.STR+second.INT/2)
	firstDefense := max(0, first.DEF/4)
	secondDefense := max(0, second.DEF/4)
	firstTurn := first.SPD >= second.SPD
	logs := []string{fmt.Sprintf("Arena: %s vs %s.", firstName, secondName)}
	for round := 1; round <= 50 && firstHP > 0 && secondHP > 0; round++ {
		if firstTurn {
			damage := max(1, firstAttack-secondDefense)
			secondHP -= damage
			logs = append(logs, fmt.Sprintf("R%d · %s strikes for %d.", round, firstName, damage))
		} else {
			damage := max(1, secondAttack-firstDefense)
			firstHP -= damage
			logs = append(logs, fmt.Sprintf("R%d · %s strikes for %d.", round, secondName, damage))
		}
		firstTurn = !firstTurn
	}
	winner := 0
	if secondHP > firstHP {
		winner = 1
	}
	return winner, append(logs, "Arena result settled without changing health, cooldowns, or equipment.")
}

func (s *WebServer) handleAbyssDuel(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Action      string `json:"action"`
		DuelID      int64  `json:"duel_id"`
		OpponentUID string `json:"opponent_uid"`
		Wager       int    `json:"wager"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	message := "Duel updated."
	var duelLog []string
	switch strings.TrimSpace(req.Action) {
	case "create":
		req.OpponentUID = strings.TrimSpace(req.OpponentUID)
		if req.OpponentUID == "" || req.OpponentUID == uid || req.Wager < 1 || req.Wager > 100 {
			writeJSON(w, map[string]any{"ok": false, "error": "choose another delver and wager 1–100 tokens"})
			return
		}
		var pending, opponentExists int
		if err := tx.QueryRow("SELECT COUNT(*) FROM abyss_duels WHERE challenger_uid=$1 AND status='pending'", uid).Scan(&pending); err != nil || pending >= 3 {
			writeJSON(w, map[string]any{"ok": false, "error": "resolve existing duel challenges first"})
			return
		}
		if err := tx.QueryRow("SELECT COUNT(*) FROM users WHERE client_uid=$1", req.OpponentUID).Scan(&opponentExists); err != nil || opponentExists != 1 {
			writeJSON(w, map[string]any{"ok": false, "error": "opponent not found"})
			return
		}
		result, err := tx.Exec("UPDATE users SET abyss_tokens=abyss_tokens-$1 WHERE client_uid=$2 AND abyss_tokens>=$1", req.Wager, uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if changed, _ := result.RowsAffected(); changed != 1 {
			writeJSON(w, map[string]any{"ok": false, "error": "not enough Abyss Tokens"})
			return
		}
		if _, err := tx.Exec(`INSERT INTO abyss_duels (challenger_uid,opponent_uid,wager_tokens)
			VALUES ($1,$2,$3)`, uid, req.OpponentUID, req.Wager); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "that duel challenge is already pending"})
			return
		}
		message = "Duel challenge created; your wager is reserved."
	case "accept", "decline", "cancel":
		action := strings.TrimSpace(req.Action)
		var challenger, opponent, status string
		var wager int
		var createdAt time.Time
		if err := tx.QueryRow(`SELECT challenger_uid,opponent_uid,wager_tokens,status,created_at
			FROM abyss_duels WHERE duel_id=$1 FOR UPDATE`, req.DuelID).Scan(&challenger, &opponent, &wager, &status, &createdAt); err != nil || status != "pending" ||
			(action == "cancel" && challenger != uid) || (action != "cancel" && opponent != uid) {
			writeJSON(w, map[string]any{"ok": false, "error": "pending duel not found"})
			return
		}
		if action != "accept" || time.Since(createdAt) > 24*time.Hour {
			if _, err := tx.Exec("UPDATE users SET abyss_tokens=abyss_tokens+$1 WHERE client_uid=$2", wager, challenger); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			_, err = tx.Exec("UPDATE abyss_duels SET status='declined',resolved_at=NOW() WHERE duel_id=$1", req.DuelID)
			message = "Duel closed; the reserved wager was returned."
			break
		}
		result, err := tx.Exec("UPDATE users SET abyss_tokens=abyss_tokens-$1 WHERE client_uid=$2 AND abyss_tokens>=$1", wager, uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if changed, _ := result.RowsAffected(); changed != 1 {
			writeJSON(w, map[string]any{"ok": false, "error": "not enough Abyss Tokens to match the wager"})
			return
		}
		var challengerName, opponentName string
		if err := tx.QueryRow("SELECT COALESCE(NULLIF(nickname,''),'Adventurer') FROM users WHERE client_uid=$1", challenger).Scan(&challengerName); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if err := tx.QueryRow("SELECT COALESCE(NULLIF(nickname,''),'Adventurer') FROM users WHERE client_uid=$1", opponent).Scan(&opponentName); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		winnerIndex, logs := abyssDuelResolve(challengerName, s.bot.abyssCombatStats(challenger), opponentName, s.bot.abyssCombatStats(opponent))
		winner := challenger
		if winnerIndex == 1 {
			winner = opponent
		}
		encoded, _ := json.Marshal(logs)
		if _, err := tx.Exec("UPDATE users SET abyss_tokens=abyss_tokens+$1 WHERE client_uid=$2", wager*2, winner); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if _, err := tx.Exec(`UPDATE abyss_duels SET status='accepted',winner_uid=$1,combat_log=$2::jsonb,resolved_at=NOW()
			WHERE duel_id=$3`, winner, string(encoded), req.DuelID); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		duelLog = logs
		message = "Arena duel settled; winner received both token stakes."
	default:
		writeJSON(w, map[string]any{"ok": false, "error": "invalid duel action"})
		return
	}
	if err != nil || tx.Commit() != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": message, "log": duelLog, "tokens": s.bot.abyssTokens(uid)})
}

func (s *WebServer) handleAbyssRaidLobby(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Action string `json:"action"`
		Code   string `json:"code"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	week, _ := abyssWeeklyBossDefinition(timeNowUTC())
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	message := "Raid lobby updated."
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	switch strings.TrimSpace(req.Action) {
	case "create":
		code, err = abyssSocialCode("R-")
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "could not create raid code"})
			return
		}
		if _, err := tx.Exec("INSERT INTO abyss_raid_lobbies (lobby_code,owner_uid,week_key) VALUES ($1,$2,$3)", code, uid, week); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "leave this week's raid lobby first"})
			return
		}
		if _, err := tx.Exec("INSERT INTO abyss_raid_members (lobby_code,week_key,client_uid) VALUES ($1,$2,$3)", code, week, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		message = "Raid lobby created."
	case "join":
		var lobbyWeek, status string
		if err := tx.QueryRow("SELECT week_key,status FROM abyss_raid_lobbies WHERE lobby_code=$1 FOR UPDATE", code).Scan(&lobbyWeek, &status); err != nil || lobbyWeek != week || status != "open" {
			writeJSON(w, map[string]any{"ok": false, "error": "raid lobby unavailable"})
			return
		}
		var count int
		if err := tx.QueryRow("SELECT COUNT(*) FROM abyss_raid_members WHERE lobby_code=$1", code).Scan(&count); err != nil || count >= abyssRaidMaxMembers {
			writeJSON(w, map[string]any{"ok": false, "error": "raid lobby is full"})
			return
		}
		if _, err := tx.Exec("INSERT INTO abyss_raid_members (lobby_code,week_key,client_uid) VALUES ($1,$2,$3)", code, week, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "you already joined a raid this week"})
			return
		}
		message = "Joined raid lobby."
	case "strike":
		var owner, status string
		if err := tx.QueryRow("SELECT owner_uid,status FROM abyss_raid_lobbies WHERE lobby_code=$1 AND week_key=$2 FOR UPDATE", code, week).Scan(&owner, &status); err != nil || owner != uid || status != "open" {
			writeJSON(w, map[string]any{"ok": false, "error": "only the open lobby owner can launch the raid"})
			return
		}
		rows, err := tx.Query(`SELECT m.client_uid FROM abyss_raid_members m
			WHERE m.lobby_code=$1 AND NOT EXISTS(SELECT 1 FROM abyss_weekly_boss_contributions c
			WHERE c.week_key=$2 AND c.client_uid=m.client_uid AND c.contribution_date=CURRENT_DATE)
			ORDER BY m.joined_at FOR UPDATE OF m`, code, week)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		var members []string
		for rows.Next() {
			var member string
			if rows.Scan(&member) == nil {
				members = append(members, member)
			}
		}
		_ = rows.Close()
		if len(members) < 2 {
			writeJSON(w, map[string]any{"ok": false, "error": "raid requires at least two eligible members"})
			return
		}
		_, bossName := abyssWeeklyBossDefinition(timeNowUTC())
		if _, err := tx.Exec(`INSERT INTO abyss_weekly_bosses (week_key,boss_name,max_hp,current_hp)
			VALUES ($1,$2,$3,$3) ON CONFLICT (week_key) DO NOTHING`, week, bossName, abyssWeeklyBossHP); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		var hp, maxHP int64
		var defeated sql.NullTime
		if err := tx.QueryRow(`SELECT current_hp,max_hp,defeated_at FROM abyss_weekly_bosses
			WHERE week_key=$1 FOR UPDATE`, week).Scan(&hp, &maxHP, &defeated); err != nil || defeated.Valid || hp <= 0 {
			writeJSON(w, map[string]any{"ok": false, "error": "weekly boss is already defeated"})
			return
		}
		var totalDamage int64
		for _, member := range members {
			damage := min(abyssWeeklyBossDamage(s.bot.abyssPlayerCR(member)), hp-totalDamage)
			if damage <= 0 {
				break
			}
			drop := abyssWeeklyBossDropFor(bossName, week, member, timeNowUTC())
			if _, err := tx.Exec(`INSERT INTO abyss_weekly_boss_contributions
				(week_key,client_uid,damage,loot_label) VALUES ($1,$2,$3,$4)`, week, member, damage, abyssWeeklyBossDropLabel(drop)); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if err := grantMaterialQ(tx, member, drop.Material, drop.Amount); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			totalDamage += damage
		}
		newHP := max(int64(0), hp-totalDamage)
		if _, err := tx.Exec(`UPDATE abyss_weekly_bosses SET current_hp=$1,
			defeated_at=CASE WHEN $1=0 THEN NOW() ELSE defeated_at END WHERE week_key=$2`, newHP, week); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if newHP == 0 {
			if _, err := tx.Exec(`UPDATE users u SET abyss_tokens=abyss_tokens+25 FROM
				(SELECT DISTINCT client_uid FROM abyss_weekly_boss_contributions WHERE week_key=$1) c
				WHERE u.client_uid=c.client_uid`, week); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
		}
		_, err = tx.Exec("UPDATE abyss_raid_lobbies SET status='resolved',resolved_at=NOW() WHERE lobby_code=$1", code)
		message = fmt.Sprintf("Raid dealt %d shared damage; every member received an individual loot roll.", totalDamage)
	default:
		writeJSON(w, map[string]any{"ok": false, "error": "invalid raid action"})
		return
	}
	if err != nil || tx.Commit() != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "code": code, "invite_url": "/abyss?raid=" + code + "#social", "msg": message})
}
