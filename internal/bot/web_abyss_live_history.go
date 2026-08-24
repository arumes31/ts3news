package bot

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type abyssLiveIdempotency struct {
	Round  int
	Action abyssLiveAction
}

type abyssLiveEvent struct {
	ID        int64                        `json:"id"`
	At        time.Time                    `json:"at"`
	Round     int                          `json:"round"`
	Phase     string                       `json:"phase"`
	Snapshots map[string]abyssLiveSnapshot `json:"snapshots"`
}

type abyssLivePersistedState struct {
	SchemaVersion int               `json:"schema_version"`
	RandomSeed    [2]uint64         `json:"random_seed"`
	Snapshot      abyssLiveSnapshot `json:"snapshot"`
	Events        []abyssLiveEvent  `json:"events"`
}

type abyssLiveReplayArchive struct {
	SessionID  string                  `json:"session_id"`
	OwnerUID   string                  `json:"owner_uid"`
	ArchivedAt time.Time               `json:"archived_at"`
	State      abyssLivePersistedState `json:"state"`
}

func (c *abyssLiveCombat) eventsAfter(uid string, eventID int64) []abyssLiveSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	events := make([]abyssLiveSnapshot, 0)
	for _, event := range c.history {
		if event.ID <= eventID {
			continue
		}
		if snapshot, ok := event.Snapshots[uid]; ok {
			events = append(events, cloneAbyssLiveSnapshot(snapshot))
		}
	}
	return events
}

func cloneAbyssLiveSnapshot(snapshot abyssLiveSnapshot) abyssLiveSnapshot {
	snapshot.Allies = append([]abyssLiveCombatantView{}, snapshot.Allies...)
	snapshot.Enemies = append([]abyssLiveCombatantView{}, snapshot.Enemies...)
	snapshot.Options = append([]abyssLiveOption{}, snapshot.Options...)
	snapshot.RecentLogs = append([]string{}, snapshot.RecentLogs...)
	if snapshot.Queued != nil {
		queued := *snapshot.Queued
		snapshot.Queued = &queued
	}
	if snapshot.Result != nil {
		result := make(map[string]any, len(snapshot.Result))
		for key, value := range snapshot.Result {
			result[key] = value
		}
		snapshot.Result = result
	}
	return snapshot
}

func (c *abyssLiveCombat) encodeReplayArchive(
	snapshot abyssLiveSnapshot,
	history []abyssLiveEvent,
) (string, error) {
	return encodeAbyssLiveReplayArchive(c.id, c.ownerUID, abyssLivePersistedState{
		SchemaVersion: abyssLiveSnapshotSchemaVersion,
		RandomSeed:    c.randomSeed,
		Snapshot:      snapshot,
		Events:        history,
	})
}

func encodeAbyssLiveReplayArchive(
	sessionID string,
	ownerUID string,
	state abyssLivePersistedState,
) (string, error) {
	archive := abyssLiveReplayArchive{
		SessionID:  sessionID,
		OwnerUID:   ownerUID,
		ArchivedAt: time.Now().UTC(),
		State:      state,
	}
	encoded, err := json.Marshal(archive)
	if err != nil {
		return "", fmt.Errorf("encoding live combat replay: %w", err)
	}
	return string(encoded), nil
}

func (c *abyssLiveCombat) archiveReplayInTx(
	ctx context.Context,
	tx *sql.Tx,
	encoded string,
	members []abyssLiveMemberPersistence,
) error {
	uids := make([]string, 0, len(members))
	for _, member := range members {
		uids = append(uids, member.uid)
	}
	return archiveAbyssLiveReplayInTx(ctx, tx, c.id, encoded, uids)
}

func archiveAbyssLiveReplayInTx(
	ctx context.Context,
	tx *sql.Tx,
	sessionID string,
	encoded string,
	uids []string,
) error {
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`,
		"abyss_live_replay_session_"+sessionID,
		encoded,
	); err != nil {
		return fmt.Errorf("archiving canonical live combat replay: %w", err)
	}
	for _, uid := range uids {
		key := "abyss_live_replay_user_" + uid + "_" + sessionID
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO app_meta (key, value) VALUES ($1, $2)
			 ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`,
			key,
			sessionID,
		); err != nil {
			return fmt.Errorf("indexing live combat replay for %s: %w", uid, err)
		}
	}
	return nil
}

func archiveRecoveredAbyssLiveReplayInTx(
	ctx context.Context,
	tx *sql.Tx,
	sessionID string,
	ownerUID string,
	stateJSON string,
	terminalSnapshot abyssLiveSnapshot,
) error {
	var state abyssLivePersistedState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return fmt.Errorf("decoding recovered live combat replay: %w", err)
	}

	rows, err := tx.QueryContext(
		ctx,
		"SELECT client_uid FROM abyss_combat_members WHERE session_id=$1 ORDER BY client_uid FOR UPDATE",
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("loading recovered live combat participants: %w", err)
	}
	uids := make([]string, 0, 2)
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			_ = rows.Close()
			return fmt.Errorf("reading recovered live combat participant: %w", err)
		}
		uids = append(uids, uid)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterating recovered live combat participants: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("closing recovered live combat participants: %w", err)
	}

	ownerSnapshot := state.Snapshot
	ownerSnapshot.SchemaVersion = abyssLiveSnapshotSchemaVersion
	ownerSnapshot.SessionID = sessionID
	ownerSnapshot.OwnerUID = ownerUID
	ownerSnapshot.PreviousDepth = terminalSnapshot.PreviousDepth
	ownerSnapshot.Phase = terminalSnapshot.Phase
	ownerSnapshot.Result = terminalSnapshot.Result
	ownerSnapshot.Version = terminalSnapshot.Version
	ownerSnapshot.OK = true
	state.SchemaVersion = abyssLiveSnapshotSchemaVersion
	if state.RandomSeed == [2]uint64{} {
		state.RandomSeed = terminalSnapshot.RandomSeed
	}
	state.Snapshot = ownerSnapshot
	if len(state.Events) == 0 || state.Events[len(state.Events)-1].ID < terminalSnapshot.Version {
		snapshots := make(map[string]abyssLiveSnapshot, len(uids))
		for _, uid := range uids {
			snapshots[uid] = cloneAbyssLiveSnapshot(terminalSnapshot)
		}
		state.Events = append(state.Events, abyssLiveEvent{
			ID:        terminalSnapshot.Version,
			At:        time.Now().UTC(),
			Round:     terminalSnapshot.Round,
			Phase:     terminalSnapshot.Phase,
			Snapshots: snapshots,
		})
	}
	encoded, err := encodeAbyssLiveReplayArchive(sessionID, ownerUID, state)
	if err != nil {
		return err
	}
	return archiveAbyssLiveReplayInTx(ctx, tx, sessionID, encoded, uids)
}
