package bot

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

type abyssCompletedRunReplayView struct {
	Version   int                   `json:"version"`
	RunID     int64                 `json:"run_id"`
	Depth     int                   `json:"depth"`
	Victory   bool                  `json:"victory"`
	Tier      string                `json:"tier"`
	StartedAt string                `json:"started_at"`
	EndedAt   string                `json:"ended_at"`
	EndReason string                `json:"end_reason"`
	AuditHash string                `json:"audit_hash"`
	RunSeed   []string              `json:"run_seed"`
	Choices   []abyssRunChoice      `json:"choices"`
	Floors    []abyssRunFloorReplay `json:"floors"`
}

type abyssRunFloorReplay struct {
	Depth         int      `json:"depth"`
	Biome         string   `json:"biome,omitempty"`
	Victory       bool     `json:"victory"`
	HP            int      `json:"hp"`
	MaxHP         int      `json:"max_hp"`
	LegendaryDrop bool     `json:"legendary_drop,omitempty"`
	Seed          []string `json:"seed"`
	Logs          []string `json:"logs"`
}

func (s *WebServer) handleAbyssRunReplay(
	w http.ResponseWriter,
	r *http.Request,
	uid string,
) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONStatus(w, http.StatusMethodNotAllowed, map[string]any{
			"ok": false, "error": "GET only",
		})
		return
	}
	runID, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil || runID < 1 {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{
			"ok": false, "error": "invalid run id",
		})
		return
	}

	view, err := s.loadAbyssRunReplay(r.Context(), uid, runID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSONStatus(w, http.StatusNotFound, map[string]any{
			"ok": false, "error": "run replay not found",
		})
		return
	}
	if err != nil {
		writeJSONStatus(w, http.StatusServiceUnavailable, map[string]any{
			"ok": false, "error": "run replay unavailable",
		})
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, map[string]any{"ok": true, "replay": view})
}

func (s *WebServer) loadAbyssRunReplay(
	ctx context.Context,
	uid string,
	runID int64,
) (abyssCompletedRunReplayView, error) {
	var raw []byte
	var hash string
	if err := s.bot.DB.QueryRowContext(
		ctx,
		"SELECT audit_hash,audit_data FROM abyss_runs WHERE id=$1 AND client_uid=$2",
		runID,
		uid,
	).Scan(&hash, &raw); err != nil {
		return abyssCompletedRunReplayView{}, err
	}
	var audit abyssCompetitionAudit
	if err := json.Unmarshal(raw, &audit); err != nil {
		return abyssCompletedRunReplayView{}, fmt.Errorf("decoding run audit: %w", err)
	}
	canonical, err := json.Marshal(audit)
	if err != nil {
		return abyssCompletedRunReplayView{}, fmt.Errorf("encoding run audit: %w", err)
	}
	digest := sha256.Sum256(canonical)
	if hash == "" || !equalAbyssAuditHash(hash, hex.EncodeToString(digest[:])) {
		return abyssCompletedRunReplayView{}, errors.New("run audit verification failed")
	}

	choices := make([]abyssRunChoice, 0, len(audit.Choices))
	for _, choice := range audit.Choices {
		choice.Kind = boundedAbyssReplayText(choice.Kind, 80)
		choice.Value = boundedAbyssReplayText(choice.Value, abyssReplayViewTextMaxRunes)
		choices = append(choices, choice)
	}
	floors := make([]abyssRunFloorReplay, 0, len(audit.Floors))
	for _, floor := range audit.Floors {
		logs := make([]string, 0, len(floor.Logs))
		for _, line := range floor.Logs {
			logs = append(logs, boundedAbyssReplayText(line, abyssReplayViewTextMaxRunes))
		}
		floors = append(floors, abyssRunFloorReplay{
			Depth:         floor.Depth,
			Biome:         boundedAbyssReplayText(floor.Biome, 120),
			Victory:       floor.Victory,
			HP:            floor.HP,
			MaxHP:         floor.MaxHP,
			LegendaryDrop: floor.LegendaryDrop,
			Seed:          abyssSeedStrings(floor.Seed),
			Logs:          logs,
		})
	}
	runSeed := []string{}
	if audit.RunSeed != nil {
		runSeed = abyssSeedStrings(*audit.RunSeed)
	}
	return abyssCompletedRunReplayView{
		Version:   audit.Version,
		RunID:     runID,
		Depth:     audit.Depth,
		Victory:   audit.Victory,
		Tier:      boundedAbyssReplayText(audit.Tier, 40),
		StartedAt: audit.StartedAt,
		EndedAt:   audit.EndedAt,
		EndReason: boundedAbyssReplayText(audit.EndReason, 80),
		AuditHash: hash,
		RunSeed:   runSeed,
		Choices:   choices,
		Floors:    floors,
	}, nil
}

func abyssSeedStrings(seed [2]uint64) []string {
	return []string{
		strconv.FormatUint(seed[0], 10),
		strconv.FormatUint(seed[1], 10),
	}
}

func equalAbyssAuditHash(left, right string) bool {
	leftBytes, leftErr := hex.DecodeString(left)
	rightBytes, rightErr := hex.DecodeString(right)
	return leftErr == nil && rightErr == nil && subtle.ConstantTimeCompare(leftBytes, rightBytes) == 1
}
