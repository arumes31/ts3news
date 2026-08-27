package bot

import (
	"net/http"
	"strings"
)

func (s *WebServer) handleAbyssGuild(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Action     string `json:"action"`
		Name       string `json:"name"`
		Tag        string `json:"tag"`
		InviteCode string `json:"invite_code"`
		Banner     string `json:"banner"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	req.Action = strings.TrimSpace(req.Action)
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	message := "Guild updated."
	switch req.Action {
	case "create":
		req.Name = strings.TrimSpace(req.Name)
		req.Tag = strings.ToUpper(strings.TrimSpace(req.Tag))
		if !abyssSocialNameValid(req.Name, 3, 32) || !abyssSocialNameValid(req.Tag, 2, 5) || strings.Contains(req.Tag, " ") {
			writeJSON(w, map[string]any{"ok": false, "error": "use a 3–32 character guild name and a 2–5 character tag"})
			return
		}
		code, codeErr := abyssSocialCode("G-")
		if codeErr != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "could not create invite code"})
			return
		}
		var guildID int64
		if err := tx.QueryRow(`INSERT INTO abyss_guilds (name,tag,owner_uid,invite_code)
			SELECT $1,$2,$3,$4 WHERE NOT EXISTS(SELECT 1 FROM abyss_guild_members WHERE client_uid=$3)
			RETURNING guild_id`, req.Name, req.Tag, uid, code).Scan(&guildID); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "guild name/tag unavailable or you already belong to a guild"})
			return
		}
		if _, err := tx.Exec(`INSERT INTO abyss_guild_members (guild_id,client_uid,role)
			VALUES ($1,$2,'owner')`, guildID, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		message = "Guild created. Invite code: " + code
	case "join":
		req.InviteCode = strings.ToUpper(strings.TrimSpace(req.InviteCode))
		var guildID int64
		var members int
		if err := tx.QueryRow(`SELECT guild_id FROM abyss_guilds
			WHERE invite_code=$1 FOR UPDATE`, req.InviteCode).Scan(&guildID); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "guild invite invalid or guild full"})
			return
		}
		if err := tx.QueryRow(`SELECT COUNT(*) FROM abyss_guild_members
			WHERE guild_id=$1`, guildID).Scan(&members); err != nil || members >= 50 {
			writeJSON(w, map[string]any{"ok": false, "error": "guild invite invalid or guild full"})
			return
		}
		if _, err := tx.Exec(`INSERT INTO abyss_guild_members (guild_id,client_uid)
			VALUES ($1,$2)`, guildID, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "leave your current guild before joining another"})
			return
		}
		message = "Joined guild."
	case "leave":
		var role string
		if err := tx.QueryRow("SELECT role FROM abyss_guild_members WHERE client_uid=$1 FOR UPDATE", uid).Scan(&role); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "you are not in a guild"})
			return
		}
		if role == "owner" {
			writeJSON(w, map[string]any{"ok": false, "error": "guild owners must disband instead of leaving"})
			return
		}
		_, err = tx.Exec("DELETE FROM abyss_guild_members WHERE client_uid=$1", uid)
		message = "Left guild."
	case "disband":
		result, deleteErr := tx.Exec("DELETE FROM abyss_guilds WHERE owner_uid=$1", uid)
		err = deleteErr
		if err == nil {
			if changed, _ := result.RowsAffected(); changed != 1 {
				writeJSON(w, map[string]any{"ok": false, "error": "only the guild owner can disband"})
				return
			}
		}
		message = "Guild disbanded."
	case "banner":
		req.Banner = strings.TrimSpace(req.Banner)
		if req.Banner != "standard" && req.Banner != "ember" && req.Banner != "tide" && req.Banner != "verdant" && req.Banner != "void" {
			writeJSON(w, map[string]any{"ok": false, "error": "unknown guild banner"})
			return
		}
		result, updateErr := tx.Exec("UPDATE abyss_guilds SET banner=$1 WHERE owner_uid=$2", req.Banner, uid)
		err = updateErr
		if err == nil {
			if changed, _ := result.RowsAffected(); changed != 1 {
				writeJSON(w, map[string]any{"ok": false, "error": "only the guild owner can change its banner"})
				return
			}
		}
		message = "Guild banner updated. Cosmetic only."
	default:
		writeJSON(w, map[string]any{"ok": false, "error": "invalid guild action"})
		return
	}
	if err != nil || tx.Commit() != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": message})
}

func (s *WebServer) handleAbyssTournamentTeam(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Action   string `json:"action"`
		Name     string `json:"name"`
		TeamCode string `json:"team_code"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	week := abyssCurrentWeek(timeNowUTC())
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	message := "Tournament team updated."
	switch strings.TrimSpace(req.Action) {
	case "create":
		req.Name = strings.TrimSpace(req.Name)
		if !abyssSocialNameValid(req.Name, 3, 24) {
			writeJSON(w, map[string]any{"ok": false, "error": "use a 3–24 character team name"})
			return
		}
		code, codeErr := abyssSocialCode("T-")
		if codeErr != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "could not create team code"})
			return
		}
		if _, err := tx.Exec(`INSERT INTO abyss_tournament_teams (week_key,team_code,owner_uid,name)
			VALUES ($1,$2,$3,$4)`, week, code, uid, req.Name); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "you already lead a team this week"})
			return
		}
		if _, err := tx.Exec(`INSERT INTO abyss_tournament_members (week_key,team_code,client_uid)
			VALUES ($1,$2,$3)`, week, code, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		message = "Tournament team created. Invite code: " + code
	case "join":
		req.TeamCode = strings.ToUpper(strings.TrimSpace(req.TeamCode))
		var lockedCode string
		if err := tx.QueryRow(`SELECT team_code FROM abyss_tournament_teams
			WHERE week_key=$1 AND team_code=$2 FOR UPDATE`, week, req.TeamCode).Scan(&lockedCode); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "team unavailable or full"})
			return
		}
		var members int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM abyss_tournament_members
			WHERE week_key=$1 AND team_code=$2`, week, req.TeamCode).Scan(&members); err != nil || members >= 3 {
			writeJSON(w, map[string]any{"ok": false, "error": "team unavailable or full"})
			return
		}
		result, err := tx.Exec(`INSERT INTO abyss_tournament_members (week_key,team_code,client_uid)
			SELECT $1,$2,$3 WHERE EXISTS(SELECT 1 FROM abyss_tournament_teams WHERE week_key=$1 AND team_code=$2)`,
			week, req.TeamCode, uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "you already joined a tournament team this week"})
			return
		}
		if changed, _ := result.RowsAffected(); changed != 1 {
			writeJSON(w, map[string]any{"ok": false, "error": "team unavailable or full"})
			return
		}
		message = "Joined tournament team. Bracket seeds lock Sunday 18:00 UTC."
	default:
		writeJSON(w, map[string]any{"ok": false, "error": "invalid tournament action"})
		return
	}
	if tx.Commit() != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": message})
}
