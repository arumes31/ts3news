package bot

import (
	"database/sql"
	"net/http"
	"strings"
)

type abyssFriendEchoCandidate struct {
	UID      string
	Nick     string
	Assists  int
	Selected bool
}

type abyssFriendEchoView struct {
	ShareEnabled bool
	SelectedUID  string
	Candidates   []abyssFriendEchoCandidate
}

type abyssEchoIdentity struct {
	UID   string
	Nick  string
	Depth int
}

func (b *Bot) abyssFriendEchoSettings(uid string) abyssFriendEchoView {
	var view abyssFriendEchoView
	_ = b.DB.QueryRow(`SELECT ghost_echo_opt_in,COALESCE(preferred_echo_uid,'')
		FROM abyss_social_profiles WHERE client_uid=$1`, uid).Scan(&view.ShareEnabled, &view.SelectedUID)

	rows, err := b.DB.Query(`SELECT u.client_uid,COALESCE(NULLIF(u.nickname,''),'Adventurer'),hb.assists
		FROM abyss_helper_bonds hb
		JOIN users u ON u.client_uid=CASE WHEN hb.uid_low=$1 THEN hb.uid_high ELSE hb.uid_low END
		JOIN abyss_social_profiles p ON p.client_uid=u.client_uid AND p.ghost_echo_opt_in=TRUE
		WHERE hb.uid_low=$1 OR hb.uid_high=$1
		ORDER BY hb.assists DESC,u.client_uid`, uid)
	if err != nil {
		return view
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var candidate abyssFriendEchoCandidate
		if rows.Scan(&candidate.UID, &candidate.Nick, &candidate.Assists) != nil {
			return view
		}
		candidate.Selected = candidate.UID == view.SelectedUID
		view.Candidates = append(view.Candidates, candidate)
	}
	return view
}

func (b *Bot) selectAbyssEchoIdentity(uid string) (abyssEchoIdentity, error) {
	var echo abyssEchoIdentity
	err := b.DB.QueryRow(`SELECT u.client_uid,COALESCE(NULLIF(u.nickname,''),'Adventurer'),r.depth
		FROM abyss_social_profiles owner
		JOIN abyss_social_profiles donor ON donor.client_uid=owner.preferred_echo_uid AND donor.ghost_echo_opt_in=TRUE
		JOIN abyss_helper_bonds hb ON (hb.uid_low=owner.client_uid AND hb.uid_high=donor.client_uid)
			OR (hb.uid_high=owner.client_uid AND hb.uid_low=donor.client_uid)
		JOIN users u ON u.client_uid=donor.client_uid
		JOIN LATERAL (SELECT depth FROM abyss_runs WHERE client_uid=donor.client_uid
			ORDER BY depth DESC,gold_banked DESC LIMIT 1) r ON TRUE
		WHERE owner.client_uid=$1`, uid).Scan(&echo.UID, &echo.Nick, &echo.Depth)
	if err == nil {
		return echo, nil
	}
	if err != sql.ErrNoRows {
		return abyssEchoIdentity{}, err
	}
	err = b.DB.QueryRow(`SELECT r.client_uid,COALESCE(NULLIF(u.nickname,''),'Adventurer'),r.depth
		FROM abyss_runs r JOIN users u ON u.client_uid=r.client_uid
		WHERE r.client_uid<>$1
		ORDER BY r.depth DESC,r.gold_banked DESC LIMIT 1`, uid).Scan(&echo.UID, &echo.Nick, &echo.Depth)
	return echo, err
}

func (s *WebServer) handleAbyssFriendEchoSettings(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ShareEnabled bool   `json:"share_enabled"`
		EchoUID      string `json:"echo_uid"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	req.EchoUID = strings.TrimSpace(req.EchoUID)
	if req.EchoUID == uid {
		writeJSON(w, map[string]any{"ok": false, "error": "choose a bonded friend"})
		return
	}
	if req.EchoUID != "" {
		var eligible bool
		err := s.bot.DB.QueryRow(`SELECT EXISTS(
			SELECT 1 FROM abyss_helper_bonds hb
			JOIN abyss_social_profiles p ON p.client_uid=$2 AND p.ghost_echo_opt_in=TRUE
			WHERE (hb.uid_low=$1 AND hb.uid_high=$2) OR (hb.uid_high=$1 AND hb.uid_low=$2))`, uid, req.EchoUID).Scan(&eligible)
		if err != nil || !eligible {
			writeJSON(w, map[string]any{"ok": false, "error": "that friend is not available for echoes"})
			return
		}
	}
	_, err := s.bot.DB.Exec(`INSERT INTO abyss_social_profiles (client_uid,ghost_echo_opt_in,preferred_echo_uid)
		VALUES ($1,$2,NULLIF($3,'')) ON CONFLICT (client_uid) DO UPDATE SET
		ghost_echo_opt_in=EXCLUDED.ghost_echo_opt_in,preferred_echo_uid=EXCLUDED.preferred_echo_uid`, uid, req.ShareEnabled, req.EchoUID)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "could not save echo settings"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": "Friend echo settings saved."})
}
