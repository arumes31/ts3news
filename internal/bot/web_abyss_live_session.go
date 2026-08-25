package bot

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"
)

const abyssLivePersistenceTimeout = 5 * time.Second

type abyssLiveMemberPersistence struct {
	uid            string
	tactic         string
	target         string
	actionJSON     any
	submittedRound int
	snapshotJSON   string
}

func newAbyssLiveSessionID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating live combat id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func abyssLiveRandomSeed(sessionID string) ([2]uint64, error) {
	decoded, err := hex.DecodeString(sessionID)
	if err != nil || len(decoded) != 16 {
		return [2]uint64{}, fmt.Errorf("invalid live combat id")
	}
	return [2]uint64{
		binary.BigEndian.Uint64(decoded[:8]),
		binary.BigEndian.Uint64(decoded[8:]),
	}, nil
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
	randomSeed, err := abyssLiveRandomSeed(id)
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
		idempotency:   make(map[string]abyssLiveIdempotency),
		actionCounts:  make(map[string]int),
		previousDepth: run.Depth,
		modifier:      modifier,
		warning:       abyssEncounterWarning(modifier),
		randomSeed:    randomSeed,
		createdAt:     time.Now(),
	}
	c.social = newAbyssLiveSocialState(s, participants, abyssPartyLootRuleFromID(s.bot.loadRunFlags(uid)["party_loot_rule"]))
	for memberUID := range participants {
		c.tactics[memberUID] = s.loadAbyssTactic(memberUID)
	}
	ownerSnapshot, initialHistory, initialMembers, err := c.capturePersistence()
	if err != nil {
		return nil, fmt.Errorf("capturing initial live combat: %w", err)
	}
	initialState, err := json.Marshal(abyssLivePersistedState{
		SchemaVersion: abyssLiveSnapshotSchemaVersion,
		RandomSeed:    c.randomSeed,
		Snapshot:      ownerSnapshot,
		Events:        initialHistory,
	})
	if err != nil {
		return nil, fmt.Errorf("encoding initial live combat: %w", err)
	}
	memberState := make(map[string]string, len(initialMembers))
	for _, member := range initialMembers {
		memberState[member.uid] = member.snapshotJSON
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
		 VALUES ($1, $2, $3, 'starting', $4)`,
		id,
		uid,
		run.Depth,
		string(initialState),
	); err != nil {
		return nil, fmt.Errorf("inserting live combat: %w", err)
	}
	for memberUID := range participants {
		if _, err := tx.Exec(
			`INSERT INTO abyss_combat_members (session_id, client_uid, tactic, state)
			 VALUES ($1, $2, $3, $4)`,
			id,
			memberUID,
			c.tactics[memberUID],
			memberState[memberUID],
		); err != nil {
			return nil, fmt.Errorf("inserting live combat member: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing live combat: %w", err)
	}
	c.mu.Lock()
	c.history = initialHistory
	c.mu.Unlock()

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
	s.abyssOps.sessionsStarted.Add(1)

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
	ctx, cancel := context.WithTimeout(context.Background(), abyssLivePersistenceTimeout)
	defer cancel()
	tx, err := s.bot.DB.BeginTx(ctx, nil)
	if err != nil {
		return abyssLiveSnapshot{}, false
	}
	defer func() { _ = tx.Rollback() }()

	var stateJSON, sessionStateJSON, ownerUID, phase, sessionID string
	var depth int
	err = tx.QueryRowContext(
		ctx,
		`SELECT m.state::text, s.state::text, s.owner_uid, s.phase, s.session_id, s.depth
		   FROM abyss_combat_sessions s
		   JOIN abyss_combat_members m ON m.session_id=s.session_id
		  WHERE m.client_uid=$1
		  ORDER BY s.updated_at DESC
		  LIMIT 1
		  FOR UPDATE OF s, m`,
		uid,
	).Scan(&stateJSON, &sessionStateJSON, &ownerUID, &phase, &sessionID, &depth)
	if err != nil {
		return abyssLiveSnapshot{}, false
	}
	var snapshot abyssLiveSnapshot
	if err := json.Unmarshal([]byte(stateJSON), &snapshot); err != nil {
		return abyssLiveSnapshot{}, false
	}
	if snapshot.SchemaVersion > abyssLiveSnapshotSchemaVersion {
		return abyssLiveSnapshot{}, false
	}
	snapshot.SchemaVersion = abyssLiveSnapshotSchemaVersion
	snapshot.SessionID = sessionID
	snapshot.OwnerUID = ownerUID
	snapshot.PreviousDepth = depth
	snapshot.Phase = phase
	if phase == "starting" || phase == "planning" || phase == "resolving" {
		if depth > 0 {
			if _, err := tx.ExecContext(
				ctx,
				"UPDATE abyss_active SET depth=$1, modifier='', event_state=NULL, last_action_at=NOW() WHERE client_uid=$2",
				depth,
				ownerUID,
			); err != nil {
				return abyssLiveSnapshot{}, false
			}
		}
		snapshot.Phase = "failed"
		snapshot.Result = map[string]any{
			"ok": false, "error": "combat was safely cancelled after a server restart; descend again",
		}
		snapshot.Version++
		snapshot.OK = true
		updatedState, marshalErr := json.Marshal(snapshot)
		if marshalErr != nil {
			return abyssLiveSnapshot{}, false
		}
		if _, err := tx.ExecContext(
			ctx,
			"UPDATE abyss_combat_sessions SET phase='failed', state=$1, version=$2, deadline=NULL, updated_at=NOW() WHERE session_id=$3",
			string(updatedState),
			snapshot.Version,
			sessionID,
		); err != nil {
			return abyssLiveSnapshot{}, false
		}
		if _, err := tx.ExecContext(
			ctx,
			"UPDATE abyss_combat_members SET state=$1, queued_action=NULL WHERE session_id=$2",
			string(updatedState),
			sessionID,
		); err != nil {
			return abyssLiveSnapshot{}, false
		}
		if err := archiveRecoveredAbyssLiveReplayInTx(
			ctx,
			tx,
			sessionID,
			ownerUID,
			sessionStateJSON,
			snapshot,
		); err != nil {
			return abyssLiveSnapshot{}, false
		}
	}
	snapshot.OK = true
	if err := tx.Commit(); err != nil {
		return abyssLiveSnapshot{}, false
	}
	return snapshot, true
}

func (c *abyssLiveCombat) persist() error {
	c.persistMu.Lock()
	defer c.persistMu.Unlock()

	ownerSnapshot, nextHistory, members, err := c.capturePersistence()
	if err != nil {
		return err
	}
	persistedState, err := json.Marshal(abyssLivePersistedState{
		SchemaVersion: abyssLiveSnapshotSchemaVersion,
		RandomSeed:    c.randomSeed,
		Snapshot:      ownerSnapshot,
		Events:        nextHistory,
	})
	if err != nil {
		return fmt.Errorf("encoding live combat session: %w", err)
	}
	var deadline any
	if !ownerSnapshot.Deadline.IsZero() {
		deadline = ownerSnapshot.Deadline
	}

	ctx, cancel := context.WithTimeout(context.Background(), abyssLivePersistenceTimeout)
	defer cancel()
	tx, err := c.server.bot.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting live combat persistence: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE abyss_combat_sessions
		    SET phase=$1, round=$2, version=$3, deadline=$4,
		        pause_reason=$5, state=$6, updated_at=NOW()
		  WHERE session_id=$7`,
		ownerSnapshot.Phase,
		ownerSnapshot.Round,
		ownerSnapshot.Version,
		deadline,
		ownerSnapshot.PauseReason,
		string(persistedState),
		c.id,
	); err != nil {
		return fmt.Errorf("persisting live combat session: %w", err)
	}
	for _, member := range members {
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE abyss_combat_members
			    SET tactic=$1, selected_target=$2, queued_action=$3,
			        submitted_round=$4, state=$5
			  WHERE session_id=$6 AND client_uid=$7`,
			member.tactic,
			member.target,
			member.actionJSON,
			member.submittedRound,
			member.snapshotJSON,
			c.id,
			member.uid,
		); err != nil {
			return fmt.Errorf("persisting live combat member: %w", err)
		}
	}
	if ownerSnapshot.Phase == "complete" || ownerSnapshot.Phase == "failed" {
		encodedReplay, err := c.encodeReplayArchive(ownerSnapshot, nextHistory)
		if err != nil {
			return err
		}
		if err := c.archiveReplayInTx(ctx, tx, encodedReplay, members); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing live combat persistence: %w", err)
	}

	c.mu.Lock()
	c.history = nextHistory
	c.mu.Unlock()
	return nil
}

func (c *abyssLiveCombat) capturePersistence() (
	abyssLiveSnapshot,
	[]abyssLiveEvent,
	[]abyssLiveMemberPersistence,
	error,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	uids := make([]string, 0, len(c.participants))
	for uid := range c.participants {
		uids = append(uids, uid)
	}
	sort.Strings(uids)
	snapshots := make(map[string]abyssLiveSnapshot, len(uids))
	members := make([]abyssLiveMemberPersistence, 0, len(uids))
	for _, uid := range uids {
		snapshot := c.snapshotForLocked(uid)
		snapshots[uid] = snapshot
		member := abyssLiveMemberPersistence{uid: uid, tactic: c.tactics[uid]}
		if action, ok := c.queued[uid]; ok {
			encoded, err := json.Marshal(action)
			if err != nil {
				return abyssLiveSnapshot{}, nil, nil, fmt.Errorf("encoding live combat action: %w", err)
			}
			member.actionJSON = string(encoded)
			member.target = action.TargetID
			member.submittedRound = action.Round
		}
		encodedSnapshot, err := json.Marshal(snapshot)
		if err != nil {
			return abyssLiveSnapshot{}, nil, nil, fmt.Errorf("encoding live combat member snapshot: %w", err)
		}
		member.snapshotJSON = string(encodedSnapshot)
		members = append(members, member)
	}

	nextHistory := append([]abyssLiveEvent{}, c.history...)
	if len(nextHistory) == 0 || nextHistory[len(nextHistory)-1].ID < c.version {
		nextHistory = append(nextHistory, abyssLiveEvent{
			ID:        c.version,
			At:        time.Now().UTC(),
			Round:     c.round,
			Phase:     c.phase,
			Snapshots: snapshots,
		})
	}
	ownerSnapshot, ok := snapshots[c.ownerUID]
	if !ok {
		ownerSnapshot = c.snapshotForLocked(c.ownerUID)
	}
	return ownerSnapshot, nextHistory, members, nil
}

func (c *abyssLiveCombat) persistOrLog(operation string) {
	if err := c.persist(); err != nil {
		log.Printf("abyss live: %s: %v", operation, err)
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
