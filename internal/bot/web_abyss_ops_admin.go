package bot

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"
)

const abyssOpsRequestLimit = 2048

type abyssBalanceDay struct {
	Date          string  `json:"date"`
	Runs          int64   `json:"runs"`
	Deaths        int64   `json:"deaths"`
	DeathRate     float64 `json:"death_rate"`
	LootDrops     int64   `json:"loot_drops"`
	FloorsCleared int64   `json:"floors_cleared"`
	DropsPerFloor float64 `json:"drops_per_floor"`
}

type abyssBalanceSnapshot struct {
	Available  bool              `json:"available"`
	WindowDays int               `json:"window_days"`
	Days       []abyssBalanceDay `json:"days"`
}

func (s *WebServer) handleAbyssOpsPage(w http.ResponseWriter, r *http.Request, _ string) {
	if s.abyssFeatures.token() == "" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	s.render(w, "abyssOps", map[string]any{"Title": "Abyss Operations", "Nav": "ops"})
}

func decodeAbyssOpsUpdate(w http.ResponseWriter, r *http.Request, update *abyssFeatureUpdate) error {
	r.Body = http.MaxBytesReader(w, r.Body, abyssOpsRequestLimit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(update); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request must contain one JSON object")
		}
		return err
	}
	return nil
}

func (s *WebServer) handleAbyssOpsUpdate(w http.ResponseWriter, r *http.Request) {
	var update abyssFeatureUpdate
	if err := decodeAbyssOpsUpdate(w, r, &update); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid configuration update"})
		return
	}
	snapshot, experimentChanged, err := s.abyssFeatures.update(update)
	if err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if experimentChanged {
		s.abyssOps.resetRewardExperiment(snapshot.RewardExperimentRevision)
	}
	writeJSON(w, map[string]any{
		"ok":                true,
		"features":          snapshot,
		"reward_experiment": s.abyssOps.rewardExperimentSnapshot(snapshot),
	})
}

func (s *WebServer) abyssBalanceSnapshot(parent context.Context) abyssBalanceSnapshot {
	snapshot := abyssBalanceSnapshot{WindowDays: 30, Days: []abyssBalanceDay{}}
	if s == nil || s.bot == nil || s.bot.DB == nil {
		return snapshot
	}
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	rows, err := s.bot.DB.QueryContext(ctx, `
		SELECT DATE_TRUNC('day', created_at), COUNT(*),
		       COUNT(*) FILTER (WHERE NOT victory),
		       COALESCE(SUM(loot_count), 0),
		       COALESCE(SUM(floors_cleared), 0)
		  FROM abyss_runs
		 WHERE created_at >= NOW() - INTERVAL '30 days'
		 GROUP BY DATE_TRUNC('day', created_at)
		 ORDER BY DATE_TRUNC('day', created_at)`)
	if err != nil {
		log.Printf("abyss ops balance query failed: %v", err)
		return snapshot
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var day time.Time
		var point abyssBalanceDay
		if err := rows.Scan(&day, &point.Runs, &point.Deaths, &point.LootDrops, &point.FloorsCleared); err != nil {
			log.Printf("abyss ops balance scan failed: %v", err)
			return abyssBalanceSnapshot{WindowDays: 30, Days: []abyssBalanceDay{}}
		}
		point.Date = day.UTC().Format("2006-01-02")
		point.DeathRate = ratio(point.Deaths, point.Runs)
		point.DropsPerFloor = ratio(point.LootDrops, point.FloorsCleared)
		snapshot.Days = append(snapshot.Days, point)
	}
	if err := rows.Err(); err != nil {
		log.Printf("abyss ops balance rows failed: %v", err)
		return abyssBalanceSnapshot{WindowDays: 30, Days: []abyssBalanceDay{}}
	}
	snapshot.Available = true
	return snapshot
}
