package bot

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const abyssPublicStatsCacheTTL = time.Minute

type abyssPublicStatsTotals struct {
	Runs          int64 `json:"runs"`
	Delvers       int64 `json:"delvers"`
	ActiveRuns    int64 `json:"active_runs"`
	FloorsCleared int64 `json:"floors_cleared"`
	DeepestFloor  int64 `json:"deepest_floor"`
	GoldBanked    int64 `json:"gold_banked"`
	BankedRuns    int64 `json:"banked_runs"`
	FailedRuns    int64 `json:"failed_runs"`
}

type abyssPublicTierStats struct {
	Tier          string `json:"tier"`
	Runs          int64  `json:"runs"`
	FloorsCleared int64  `json:"floors_cleared"`
	DeepestFloor  int64  `json:"deepest_floor"`
}

type abyssPublicStatsSnapshot struct {
	OK          bool                   `json:"ok"`
	Version     int                    `json:"version"`
	GeneratedAt time.Time              `json:"generated_at"`
	Season      string                 `json:"season"`
	Totals      abyssPublicStatsTotals `json:"totals"`
	Tiers       []abyssPublicTierStats `json:"tiers"`
}

type abyssPublicStatsCache struct {
	mu        sync.RWMutex
	snapshot  abyssPublicStatsSnapshot
	expiresAt time.Time
}

func (c *abyssPublicStatsCache) get(now time.Time) (abyssPublicStatsSnapshot, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.expiresAt.IsZero() || !now.Before(c.expiresAt) {
		return abyssPublicStatsSnapshot{}, false
	}
	return cloneAbyssPublicStats(c.snapshot), true
}

func (c *abyssPublicStatsCache) put(snapshot abyssPublicStatsSnapshot, expiresAt time.Time) {
	c.mu.Lock()
	c.snapshot = cloneAbyssPublicStats(snapshot)
	c.expiresAt = expiresAt
	c.mu.Unlock()
}

func cloneAbyssPublicStats(snapshot abyssPublicStatsSnapshot) abyssPublicStatsSnapshot {
	snapshot.Tiers = append([]abyssPublicTierStats(nil), snapshot.Tiers...)
	return snapshot
}

func (s *WebServer) handleAbyssPublicStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeJSONStatus(w, http.StatusMethodNotAllowed, map[string]any{
			"ok": false, "error": "GET or HEAD only",
		})
		return
	}

	now := time.Now().UTC()
	snapshot, err := s.loadAbyssPublicStats(r.Context(), now)
	if err != nil {
		slog.Error("load public Abyss stats", "error", err)
		writeJSONStatus(w, http.StatusServiceUnavailable, map[string]any{
			"ok": false, "error": "Abyss stats are temporarily unavailable",
		})
		return
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		slog.Error("encode public Abyss stats", "error", err)
		writeJSONStatus(w, http.StatusInternalServerError, map[string]any{
			"ok": false, "error": "Abyss stats are temporarily unavailable",
		})
		return
	}
	hash := sha256.Sum256(payload)
	etag := `"` + hex.EncodeToString(hash[:8]) + `"`

	w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("ETag", etag)
	w.Header().Set("Vary", "Accept-Encoding")
	if abyssPublicStatsETagMatches(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	if _, err := w.Write(append(payload, '\n')); err != nil {
		slog.Debug("write public Abyss stats", "error", err)
	}
}

func abyssPublicStatsETagMatches(header, etag string) bool {
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == etag {
			return true
		}
	}
	return false
}

func (s *WebServer) loadAbyssPublicStats(
	ctx context.Context,
	now time.Time,
) (abyssPublicStatsSnapshot, error) {
	if snapshot, ok := s.abyssPublicStats.get(now); ok {
		return snapshot, nil
	}
	if s.bot == nil || s.bot.DB == nil {
		return abyssPublicStatsSnapshot{}, fmt.Errorf("abyss database is unavailable")
	}

	tx, err := s.bot.DB.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return abyssPublicStatsSnapshot{}, fmt.Errorf("begin public stats snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	snapshot := abyssPublicStatsSnapshot{
		OK:          true,
		Version:     1,
		GeneratedAt: now.UTC().Truncate(time.Second),
		Season:      abyssSeasonLabelAt(now),
	}
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*), COUNT(DISTINCT client_uid),
		       COALESCE(SUM(floors_cleared), 0), COALESCE(MAX(depth), 0),
		       COALESCE(SUM(gold_banked), 0),
		       COUNT(*) FILTER (WHERE victory),
		       COUNT(*) FILTER (WHERE NOT victory),
		       (SELECT COUNT(*) FROM abyss_active)
		FROM abyss_runs`).Scan(
		&snapshot.Totals.Runs,
		&snapshot.Totals.Delvers,
		&snapshot.Totals.FloorsCleared,
		&snapshot.Totals.DeepestFloor,
		&snapshot.Totals.GoldBanked,
		&snapshot.Totals.BankedRuns,
		&snapshot.Totals.FailedRuns,
		&snapshot.Totals.ActiveRuns,
	)
	if err != nil {
		return abyssPublicStatsSnapshot{}, fmt.Errorf("query public stats totals: %w", err)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT COALESCE(tier, 'normal'), COUNT(*),
		       COALESCE(SUM(floors_cleared), 0), COALESCE(MAX(depth), 0)
		FROM abyss_runs
		GROUP BY COALESCE(tier, 'normal')`)
	if err != nil {
		return abyssPublicStatsSnapshot{}, fmt.Errorf("query public stats tiers: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var tier abyssPublicTierStats
		if err := rows.Scan(&tier.Tier, &tier.Runs, &tier.FloorsCleared, &tier.DeepestFloor); err != nil {
			return abyssPublicStatsSnapshot{}, fmt.Errorf("scan public stats tier: %w", err)
		}
		snapshot.Tiers = append(snapshot.Tiers, tier)
	}
	if err := rows.Err(); err != nil {
		return abyssPublicStatsSnapshot{}, fmt.Errorf("iterate public stats tiers: %w", err)
	}
	if err := rows.Close(); err != nil {
		return abyssPublicStatsSnapshot{}, fmt.Errorf("close public stats tiers: %w", err)
	}
	sortAbyssPublicTierStats(snapshot.Tiers)
	if err := tx.Commit(); err != nil {
		return abyssPublicStatsSnapshot{}, fmt.Errorf("commit public stats snapshot: %w", err)
	}

	s.abyssPublicStats.put(snapshot, now.Add(abyssPublicStatsCacheTTL))
	return cloneAbyssPublicStats(snapshot), nil
}

func sortAbyssPublicTierStats(tiers []abyssPublicTierStats) {
	order := make(map[string]int, len(abyssTierOrder))
	for index, tier := range abyssTierOrder {
		order[tier] = index
	}
	sort.Slice(tiers, func(i, j int) bool {
		left, leftKnown := order[tiers[i].Tier]
		right, rightKnown := order[tiers[j].Tier]
		if leftKnown != rightKnown {
			return leftKnown
		}
		if leftKnown {
			return left < right
		}
		return tiers[i].Tier < tiers[j].Tier
	})
}
