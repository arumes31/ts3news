package bot

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func newAbyssLiveSessionID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating live combat id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func (s *WebServer) loadAbyssTactic(uid string) string {
	var tactic string
	err := s.bot.DB.QueryRow(
		"SELECT value FROM app_meta WHERE key=$1",
		"abyss_live_tactic_"+uid,
	).Scan(&tactic)
	if err != nil {
		return "balanced"
	}
	return normalizeAbyssTactic(tactic)
}

func (s *WebServer) startAbyssLiveCombat(
	uid string,
	run abyssRun,
	depth int,
	tier abyssTier,
	modifier string,
	focus string,
) (*abyssLiveCombat, error) {
	id, err := newAbyssLiveSessionID()
	if err != nil {
		return nil, err
	}

	participants := map[string]bool{uid: true}
	var coopUID string
	_ = s.bot.DB.QueryRow(
		"SELECT COALESCE(coop_uid, '') FROM abyss_active WHERE client_uid=$1",
		uid,
	).Scan(&coopUID)
	if coopUID != "" {
		participants[coopUID] = true
	}

	c := &abyssLiveCombat{
		server:        s,
		id:            id,
		ownerUID:      uid,
		participants:  participants,
		tactics:       make(map[string]string, len(participants)),
		phase:         "starting",
		options:       make(map[string][]abyssLiveOption, len(participants)),
		queued:        make(map[string]abyssLiveAction, len(participants)),
		idempotency:   make(map[string]bool),
		previousDepth: run.Depth,
	}
	for memberUID := range participants {
		c.tactics[memberUID] = s.loadAbyssTactic(memberUID)
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("starting live combat transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for memberUID := range participants {
		if _, err := tx.Exec(
			"DELETE FROM abyss_combat_sessions WHERE session_id IN (SELECT session_id FROM abyss_combat_members WHERE client_uid=$1)",
			memberUID,
		); err != nil {
			return nil, fmt.Errorf("clearing previous live combat: %w", err)
		}
	}
	if _, err := tx.Exec(
		`INSERT INTO abyss_combat_sessions (session_id, owner_uid, depth, phase, state)
		 VALUES ($1, $2, $3, 'starting', '{}'::jsonb)`,
		id,
		uid,
		run.Depth,
	); err != nil {
		return nil, fmt.Errorf("inserting live combat: %w", err)
	}
	for memberUID := range participants {
		if _, err := tx.Exec(
			`INSERT INTO abyss_combat_members (session_id, client_uid, tactic)
			 VALUES ($1, $2, $3)`,
			id,
			memberUID,
			c.tactics[memberUID],
		); err != nil {
			return nil, fmt.Errorf("inserting live combat member: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing live combat: %w", err)
	}

	for memberUID := range participants {
		if old, ok := s.liveCombatByUID.Load(memberUID); ok {
			oldID, isString := old.(string)
			if isString {
				s.liveCombats.Delete(oldID)
			}
			s.liveCombatByUID.Delete(memberUID)
		}
	}
	s.liveCombats.Store(id, c)
	for memberUID := range participants {
		s.liveCombatByUID.Store(memberUID, id)
	}
	c.persist()

	go func() {
		res, fightErr := s.bot.fightAbyssFloorLive(uid, depth, tier, modifier, focus, c)
		if fightErr != nil {
			_, _ = s.bot.DB.Exec(
				"UPDATE abyss_active SET depth=$1, modifier='', event_state=NULL, last_action_at=NOW() WHERE client_uid=$2",
				run.Depth,
				uid,
			)
			c.complete(map[string]any{"ok": false, "error": "combat"})
			return
		}
		c.complete(s.finishDescendData(uid, run, depth, run.Escrow, tier, res, modifier, focus))
	}()

	return c, nil
}

func (s *WebServer) liveCombatForUID(uid string) (*abyssLiveCombat, bool) {
	value, ok := s.liveCombatByUID.Load(uid)
	if !ok {
		return nil, false
	}
	sessionID, ok := value.(string)
	if !ok {
		return nil, false
	}
	combatValue, ok := s.liveCombats.Load(sessionID)
	if !ok {
		return nil, false
	}
	return combatValue.(*abyssLiveCombat), true
}

func (s *WebServer) persistedAbyssLiveSnapshot(uid string) (abyssLiveSnapshot, bool) {
	var stateJSON, ownerUID, phase, sessionID string
	var depth int
	err := s.bot.DB.QueryRow(
		`SELECT m.state::text, s.owner_uid, s.phase, s.session_id, s.depth
		   FROM abyss_combat_sessions s
		   JOIN abyss_combat_members m ON m.session_id=s.session_id
		  WHERE m.client_uid=$1
		  ORDER BY s.updated_at DESC
		  LIMIT 1`,
		uid,
	).Scan(&stateJSON, &ownerUID, &phase, &sessionID, &depth)
	if err != nil {
		return abyssLiveSnapshot{}, false
	}
	var snapshot abyssLiveSnapshot
	if err := json.Unmarshal([]byte(stateJSON), &snapshot); err != nil {
		return abyssLiveSnapshot{}, false
	}
	if phase == "starting" || phase == "planning" || phase == "resolving" {
		if depth > 0 {
			_, _ = s.bot.DB.Exec(
				"UPDATE abyss_active SET depth=$1, modifier='', event_state=NULL, last_action_at=NOW() WHERE client_uid=$2",
				depth,
				ownerUID,
			)
		}
		snapshot.Phase = "failed"
		snapshot.Result = map[string]any{
			"ok": false, "error": "combat was safely cancelled after a server restart; descend again",
		}
		snapshot.Version++
		updatedState, marshalErr := json.Marshal(snapshot)
		if marshalErr == nil {
			_, _ = s.bot.DB.Exec(
				"UPDATE abyss_combat_sessions SET phase='failed', state=$1, version=$2, deadline=NULL, updated_at=NOW() WHERE session_id=$3",
				string(updatedState),
				snapshot.Version,
				sessionID,
			)
			_, _ = s.bot.DB.Exec(
				"UPDATE abyss_combat_members SET state=$1, queued_action=NULL WHERE session_id=$2 AND client_uid=$3",
				string(updatedState),
				sessionID,
				uid,
			)
		}
	}
	snapshot.OK = true
	return snapshot, true
}

func (c *abyssLiveCombat) persist() {
	snapshot := c.snapshotFor(c.ownerUID)
	state, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	var deadline any
	if !snapshot.Deadline.IsZero() {
		deadline = snapshot.Deadline
	}
	_, _ = c.server.bot.DB.Exec(
		`UPDATE abyss_combat_sessions
		    SET phase=$1, round=$2, version=$3, deadline=$4,
		        pause_reason=$5, state=$6, updated_at=NOW()
		  WHERE session_id=$7`,
		snapshot.Phase,
		snapshot.Round,
		snapshot.Version,
		deadline,
		snapshot.PauseReason,
		string(state),
		c.id,
	)
	c.mu.Lock()
	members := make(map[string]struct {
		tactic string
		action *abyssLiveAction
	}, len(c.participants))
	for uid := range c.participants {
		member := struct {
			tactic string
			action *abyssLiveAction
		}{tactic: c.tactics[uid]}
		if action, ok := c.queued[uid]; ok {
			copyAction := action
			member.action = &copyAction
		}
		members[uid] = member
	}
	c.mu.Unlock()
	for uid, member := range members {
		var actionJSON any
		var target string
		var submittedRound int
		if member.action != nil {
			if encoded, err := json.Marshal(member.action); err == nil {
				actionJSON = string(encoded)
			}
			target = member.action.TargetID
			submittedRound = member.action.Round
		}
		memberSnapshot, snapshotErr := json.Marshal(c.snapshotFor(uid))
		if snapshotErr != nil {
			continue
		}
		_, _ = c.server.bot.DB.Exec(
			`UPDATE abyss_combat_members
			    SET tactic=$1, selected_target=$2, queued_action=$3,
			        submitted_round=$4, state=$5
			  WHERE session_id=$6 AND client_uid=$7`,
			member.tactic,
			target,
			actionJSON,
			submittedRound,
			string(memberSnapshot),
			c.id,
			uid,
		)
	}
}

func (c *abyssLiveCombat) setTactic(uid, tactic string) error {
	tactic = normalizeAbyssTactic(tactic)
	c.mu.Lock()
	if !c.participants[uid] {
		c.mu.Unlock()
		return errAbyssLiveNotFound
	}
	c.tactics[uid] = tactic
	c.version++
	c.mu.Unlock()
	_, err := c.server.bot.DB.Exec(
		`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`,
		"abyss_live_tactic_"+uid,
		tactic,
	)
	if err == nil {
		_, err = c.server.bot.DB.Exec(
			"UPDATE abyss_combat_members SET tactic=$1 WHERE session_id=$2 AND client_uid=$3",
			tactic,
			c.id,
			uid,
		)
	}
	return err
}
