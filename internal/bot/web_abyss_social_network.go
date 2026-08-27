package bot

import (
	"database/sql"
	"net/http"
	"strings"
)

func (s *WebServer) handleAbyssFriend(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Action string `json:"action"`
		UID    string `json:"uid"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	req.UID = strings.TrimSpace(req.UID)
	low, high, ok := abyssPair(uid, req.UID)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "choose another player by exact user ID"})
		return
	}
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var target string
	if err := tx.QueryRowContext(r.Context(), "SELECT client_uid FROM users WHERE client_uid=$1 FOR UPDATE", req.UID).Scan(&target); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "player not found"})
		return
	}
	switch strings.TrimSpace(req.Action) {
	case "request":
		_, err = tx.ExecContext(r.Context(), `INSERT INTO abyss_friendships (uid_low,uid_high,requested_by)
			VALUES ($1,$2,$3) ON CONFLICT (uid_low,uid_high) DO NOTHING`, low, high, uid)
	case "accept":
		result, updateErr := tx.ExecContext(r.Context(), `UPDATE abyss_friendships SET accepted_at=NOW()
			WHERE uid_low=$1 AND uid_high=$2 AND requested_by<>$3 AND accepted_at IS NULL`, low, high, uid)
		err = updateErr
		if err == nil {
			if changed, _ := result.RowsAffected(); changed != 1 {
				writeJSON(w, map[string]any{"ok": false, "error": "friend request not found"})
				return
			}
		}
	case "remove":
		_, err = tx.ExecContext(r.Context(), "DELETE FROM abyss_friendships WHERE uid_low=$1 AND uid_high=$2", low, high)
	default:
		writeJSON(w, map[string]any{"ok": false, "error": "invalid friend action"})
		return
	}
	if err != nil || tx.Commit() != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": "Friend network updated."})
}

func (s *WebServer) handleAbyssFriendCheer(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		UID string `json:"uid"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	req.UID = strings.TrimSpace(req.UID)
	low, high, ok := abyssPair(uid, req.UID)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "choose a friend"})
		return
	}
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var friends bool
	if err := tx.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM abyss_friendships
		WHERE uid_low=$1 AND uid_high=$2 AND accepted_at IS NOT NULL)`, low, high).Scan(&friends); err != nil || !friends {
		writeJSON(w, map[string]any{"ok": false, "error": "daily cheers require an accepted friend"})
		return
	}
	result, err := tx.ExecContext(r.Context(), `INSERT INTO abyss_friend_cheers
		(sender_uid,recipient_uid,cheer_date,remaining_fights) VALUES ($1,$2,CURRENT_DATE,$3)
		ON CONFLICT (sender_uid,cheer_date) DO NOTHING`, uid, req.UID, abyssFriendCheerFights)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "today's cheer was already sent"})
		return
	}
	if _, err := tx.ExecContext(r.Context(), `INSERT INTO abyss_social_notifications (client_uid,kind,message)
		VALUES ($1,'friend_cheer','A friend sent +5% combat stats for your next 3 fights.')`, req.UID); err != nil || tx.Commit() != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": "Daily cheer sent: +5% stats for 3 fights."})
}

func (s *WebServer) handleAbyssMentor(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		MenteeUID string `json:"mentee_uid"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	req.MenteeUID = strings.TrimSpace(req.MenteeUID)
	if req.MenteeUID == "" || req.MenteeUID == uid {
		writeJSON(w, map[string]any{"ok": false, "error": "choose a new delver"})
		return
	}
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var mentorDepth, menteeDepth int
	if err := tx.QueryRowContext(r.Context(), "SELECT abyss_best_depth FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&mentorDepth); err != nil || mentorDepth < 25 {
		writeJSON(w, map[string]any{"ok": false, "error": "mentors must have reached depth 25"})
		return
	}
	if err := tx.QueryRowContext(r.Context(), "SELECT abyss_best_depth FROM users WHERE client_uid=$1 FOR UPDATE", req.MenteeUID).Scan(&menteeDepth); err != nil || menteeDepth > 10 {
		writeJSON(w, map[string]any{"ok": false, "error": "mentees must be at depth 10 or below"})
		return
	}
	result, err := tx.ExecContext(r.Context(), `INSERT INTO abyss_mentor_pairs (mentor_uid,mentee_uid)
		VALUES ($1,$2) ON CONFLICT (mentee_uid) DO NOTHING`, uid, req.MenteeUID)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "that delver already has a mentor"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": "Mentor bond formed. Shared clears grant both delvers bonus tokens."})
}

func (s *WebServer) handleAbyssReferral(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
	if !strings.HasPrefix(req.Code, "ABYSS-") || len(req.Code) != 16 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid referral code"})
		return
	}
	var referrer string
	if err := s.bot.DB.QueryRowContext(r.Context(), `SELECT client_uid FROM abyss_referral_codes
		WHERE code=$1 AND client_uid<>$2`, req.Code, uid).Scan(&referrer); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "referral code not found"})
		return
	}
	result, err := s.bot.DB.ExecContext(r.Context(), `INSERT INTO abyss_referrals (referrer_uid,referred_uid,referral_code)
		SELECT $1,$2,$3 FROM users WHERE client_uid=$2 AND abyss_best_depth<=5 AND first_seen>NOW()-INTERVAL '30 days'
		ON CONFLICT (referred_uid) DO NOTHING`, referrer, uid, req.Code)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "referral eligibility ended or was already claimed"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": "Referral linked. Both delvers earn 20 tokens when you reach depth 5."})
}

func acceptedAbyssFriendsQ(queryer interface {
	QueryRow(string, ...any) *sql.Row
}, uid, other string) bool {
	low, high, ok := abyssPair(uid, other)
	if !ok {
		return false
	}
	var accepted bool
	_ = queryer.QueryRow(`SELECT EXISTS(SELECT 1 FROM abyss_friendships
		WHERE uid_low=$1 AND uid_high=$2 AND accepted_at IS NOT NULL)`, low, high).Scan(&accepted)
	return accepted
}
