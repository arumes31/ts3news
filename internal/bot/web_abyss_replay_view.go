package bot

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	abyssReplaySessionIDMax        = 128
	abyssReplayRequestMaxBytes     = 4 << 10
	abyssReplayArchiveMaxBytes     = 8 << 20
	abyssReplayViewMaxFrames       = 300
	abyssReplayViewMaxLogsPerFrame = 12
	abyssReplayViewTextMaxRunes    = 500
	abyssReplayViewMaxUnits        = 64
	abyssReplayViewMaxUnitHP       = int64(1_000_000_000_000_000)
)

var (
	errAbyssReplayNotFound    = errors.New("abyss replay not found")
	errAbyssReplayUnavailable = errors.New("abyss replay unavailable")
	abyssReplayHTMLTagPattern = regexp.MustCompile(`<[^>]*>`)
)

type abyssReplaySideSummary struct {
	Alive int   `json:"alive"`
	Units int   `json:"units"`
	HP    int64 `json:"hp"`
	MaxHP int64 `json:"max_hp"`
}

type abyssReplayViewFrame struct {
	EventID int64                  `json:"event_id"`
	At      time.Time              `json:"at"`
	Round   int                    `json:"round"`
	Phase   string                 `json:"phase"`
	Allies  abyssReplaySideSummary `json:"allies"`
	Enemies abyssReplaySideSummary `json:"enemies"`
	Logs    []string               `json:"logs"`
}

type abyssReplayView struct {
	Version     int                    `json:"version"`
	SessionID   string                 `json:"session_id"`
	ArchivedAt  time.Time              `json:"archived_at"`
	RandomSeed  [2]uint64              `json:"random_seed"`
	Truncated   bool                   `json:"truncated"`
	TotalEvents int                    `json:"total_events"`
	Frames      []abyssReplayViewFrame `json:"frames"`
}

type abyssReplayRequest struct {
	SessionID string `json:"session_id"`
	Code      string `json:"code"`
	Ghost     bool   `json:"ghost"`
	View      bool   `json:"view"`
}

func decodeAbyssReplayRequest(w http.ResponseWriter, r *http.Request) (abyssReplayRequest, error) {
	r.Body = http.MaxBytesReader(w, r.Body, abyssReplayRequestMaxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request abyssReplayRequest
	if err := decoder.Decode(&request); err != nil {
		return abyssReplayRequest{}, fmt.Errorf("decoding replay request: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err == nil {
		return abyssReplayRequest{}, errors.New("replay request contains multiple JSON values")
	} else if !errors.Is(err, io.EOF) {
		return abyssReplayRequest{}, fmt.Errorf("decoding replay request suffix: %w", err)
	}
	return request, nil
}

func validAbyssReplaySessionID(sessionID string) bool {
	if sessionID == "" || len(sessionID) > abyssReplaySessionIDMax {
		return false
	}
	for _, char := range sessionID {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_' {
			continue
		}
		return false
	}
	return true
}

func (s *WebServer) loadOwnedAbyssReplay(
	ctx context.Context,
	uid string,
	sessionID string,
) (abyssLiveReplayArchive, error) {
	if !validAbyssReplaySessionID(sessionID) {
		return abyssLiveReplayArchive{}, errAbyssReplayNotFound
	}
	if s.bot == nil || s.bot.DB == nil {
		return abyssLiveReplayArchive{}, errAbyssReplayUnavailable
	}

	var owned string
	err := s.bot.DB.QueryRowContext(
		ctx,
		"SELECT value FROM app_meta WHERE key=$1",
		"abyss_live_replay_user_"+uid+"_"+sessionID,
	).Scan(&owned)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && owned != sessionID) {
		return abyssLiveReplayArchive{}, errAbyssReplayNotFound
	}
	if err != nil {
		return abyssLiveReplayArchive{}, fmt.Errorf("loading replay ownership: %w", err)
	}

	var raw string
	err = s.bot.DB.QueryRowContext(
		ctx,
		"SELECT value FROM app_meta WHERE key=$1",
		"abyss_live_replay_session_"+sessionID,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return abyssLiveReplayArchive{}, errAbyssReplayNotFound
	}
	if err != nil {
		return abyssLiveReplayArchive{}, fmt.Errorf("loading replay archive: %w", err)
	}
	if len(raw) == 0 || len(raw) > abyssReplayArchiveMaxBytes {
		return abyssLiveReplayArchive{}, errAbyssReplayUnavailable
	}

	var archive abyssLiveReplayArchive
	if err := json.Unmarshal([]byte(raw), &archive); err != nil {
		return abyssLiveReplayArchive{}, fmt.Errorf("%w: decoding archive: %w", errAbyssReplayUnavailable, err)
	}
	if archive.SessionID != sessionID || archive.State.SchemaVersion < 1 ||
		archive.State.SchemaVersion > abyssLiveSnapshotSchemaVersion {
		return abyssLiveReplayArchive{}, errAbyssReplayUnavailable
	}
	return archive, nil
}

func buildAbyssReplayView(
	archive abyssLiveReplayArchive,
	uid string,
) abyssReplayView {
	events := archive.State.Events
	start := max(0, len(events)-abyssReplayViewMaxFrames)
	view := abyssReplayView{
		Version:     1,
		SessionID:   archive.SessionID,
		ArchivedAt:  archive.ArchivedAt,
		RandomSeed:  archive.State.RandomSeed,
		Truncated:   start > 0,
		TotalEvents: len(events),
		Frames:      make([]abyssReplayViewFrame, 0, min(len(events), abyssReplayViewMaxFrames)),
	}
	for _, event := range events[start:] {
		snapshot, ok := event.Snapshots[uid]
		if !ok {
			continue
		}
		view.Frames = append(view.Frames, abyssReplayFrame(event, snapshot))
	}
	if len(view.Frames) == 0 && archive.OwnerUID == uid {
		snapshot := archive.State.Snapshot
		view.Frames = append(view.Frames, abyssReplayFrame(abyssLiveEvent{
			ID: snapshot.Version, At: archive.ArchivedAt, Round: snapshot.Round, Phase: snapshot.Phase,
		}, snapshot))
	}
	return view
}

func abyssReplayFrame(event abyssLiveEvent, snapshot abyssLiveSnapshot) abyssReplayViewFrame {
	logs := snapshot.RecentLogs
	if len(logs) > abyssReplayViewMaxLogsPerFrame {
		logs = logs[len(logs)-abyssReplayViewMaxLogsPerFrame:]
	}
	boundedLogs := make([]string, 0, len(logs))
	for _, line := range logs {
		if line = boundedAbyssReplayText(line, abyssReplayViewTextMaxRunes); line != "" {
			boundedLogs = append(boundedLogs, line)
		}
	}
	return abyssReplayViewFrame{
		EventID: event.ID,
		At:      event.At,
		Round:   max(0, event.Round),
		Phase:   boundedAbyssReplayText(event.Phase, 32),
		Allies:  summarizeAbyssReplaySide(snapshot.Allies),
		Enemies: summarizeAbyssReplaySide(snapshot.Enemies),
		Logs:    boundedLogs,
	}
}

func summarizeAbyssReplaySide(units []abyssLiveCombatantView) abyssReplaySideSummary {
	if len(units) > abyssReplayViewMaxUnits {
		units = units[:abyssReplayViewMaxUnits]
	}
	summary := abyssReplaySideSummary{Units: len(units)}
	for _, unit := range units {
		hp := min(abyssReplayViewMaxUnitHP, max(int64(0), int64(unit.HP)))
		maxHP := min(abyssReplayViewMaxUnitHP, max(int64(0), int64(unit.MaxHP)))
		summary.HP = min(abyssReplayViewMaxUnitHP, summary.HP+hp)
		summary.MaxHP = min(abyssReplayViewMaxUnitHP, summary.MaxHP+maxHP)
		if hp > 0 {
			summary.Alive++
		}
	}
	return summary
}

func boundedAbyssReplayText(value string, limit int) string {
	value = html.UnescapeString(abyssReplayHTMLTagPattern.ReplaceAllString(value, ""))
	value = strings.Map(func(char rune) rune {
		if char < 0x20 || char == 0x7f {
			return ' '
		}
		return char
	}, strings.TrimSpace(value))
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > limit {
		value = string(runes[:limit])
	}
	return value
}
