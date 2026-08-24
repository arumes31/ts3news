package bot

import (
	"crypto/subtle"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ts3news/internal/content"
)

// abyssFeatureConfig keeps risky live-combat additions independently
// reversible. Rollout assignment is stable per player, so refreshing a run
// cannot move somebody between cohorts.
type abyssFeatureConfig struct {
	liveActions bool
	social      bool
	rollout     int
	opsToken    string
}

func newAbyssFeatureConfig(b *Bot) abyssFeatureConfig {
	cfg := abyssFeatureConfig{liveActions: true, social: true, rollout: 100}
	if b != nil && b.Cfg != nil {
		cfg.liveActions = b.Cfg.AbyssLiveActions
		cfg.social = b.Cfg.AbyssSocial
		cfg.rollout = min(100, max(0, b.Cfg.AbyssLiveRolloutPercent))
		cfg.opsToken = b.Cfg.AbyssOpsToken
	}
	return cfg
}

func (c abyssFeatureConfig) enabled(feature, uid string) bool {
	switch feature {
	case "social":
		if !c.social {
			return false
		}
	case "live_actions":
		if !c.liveActions {
			return false
		}
	default:
		return false
	}
	if c.rollout >= 100 {
		return true
	}
	if c.rollout <= 0 {
		return false
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(uid))
	return int(h.Sum32()%100) < c.rollout
}

type abyssOpsOutcome struct {
	Runs int64 `json:"runs"`
	Wins int64 `json:"wins"`
}

type abyssOpsMetrics struct {
	sessionsStarted   atomic.Int64
	sessionsCompleted atomic.Int64
	automaticActions  atomic.Int64
	manualActions     atomic.Int64
	sseConnections    atomic.Int64
	requestCount      atomic.Int64
	requestNanos      atomic.Int64
	requestMaxNanos   atomic.Int64
	planningCount     atomic.Int64
	planningNanos     atomic.Int64
	resolutionCount   atomic.Int64
	resolutionNanos   atomic.Int64
	anomalies         atomic.Int64
	rewardAnomalies   atomic.Int64
	damageAnomalies   atomic.Int64
	economyAnomalies  atomic.Int64

	mu         sync.Mutex
	depthBands map[string]abyssOpsOutcome
	builds     map[string]abyssOpsOutcome
}

func (m *abyssOpsMetrics) observeAction(automatic bool) {
	if automatic {
		m.automaticActions.Add(1)
		return
	}
	m.manualActions.Add(1)
}

func (m *abyssOpsMetrics) observeRequest(d time.Duration) {
	m.requestCount.Add(1)
	m.requestNanos.Add(d.Nanoseconds())
	for old := m.requestMaxNanos.Load(); d.Nanoseconds() > old; old = m.requestMaxNanos.Load() {
		if m.requestMaxNanos.CompareAndSwap(old, d.Nanoseconds()) {
			break
		}
	}
}

func (m *abyssOpsMetrics) observePlanning(d time.Duration) {
	m.planningCount.Add(1)
	m.planningNanos.Add(d.Nanoseconds())
}

func (m *abyssOpsMetrics) observeResolution(d time.Duration) {
	m.resolutionCount.Add(1)
	m.resolutionNanos.Add(d.Nanoseconds())
}

func abyssDepthBand(depth int) string {
	switch {
	case depth < 10:
		return "1-9"
	case depth < 25:
		return "10-24"
	case depth < 50:
		return "25-49"
	default:
		return "50+"
	}
}

func dominantAbyssAction(counts map[string]int) string {
	best, bestCount := "mixed", 0
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if counts[key] > bestCount {
			best, bestCount = key, counts[key]
		}
	}
	return best
}

func (m *abyssOpsMetrics) observeCompletion(c *abyssLiveCombat, result map[string]any) {
	m.sessionsCompleted.Add(1)
	c.mu.Lock()
	depth := c.previousDepth + 1
	build := dominantAbyssAction(c.actionCounts)
	c.mu.Unlock()
	if value, ok := result["depth"].(int); ok {
		depth = value
	}
	victory, _ := result["victory"].(bool)
	m.mu.Lock()
	if m.depthBands == nil {
		m.depthBands = make(map[string]abyssOpsOutcome)
	}
	m.depthBands[abyssDepthBand(depth)] = addAbyssOutcome(m.depthBands[abyssDepthBand(depth)], victory)
	if m.builds == nil {
		m.builds = make(map[string]abyssOpsOutcome)
	}
	m.builds[build] = addAbyssOutcome(m.builds[build], victory)
	m.mu.Unlock()

}

func addAbyssOutcome(out abyssOpsOutcome, victory bool) abyssOpsOutcome {
	out.Runs++
	if victory {
		out.Wins++
	}
	return out
}

// abyssEconomyAnomaly deliberately uses a generous ceiling: it alerts on
// duplication/exploit-scale rewards without flagging ordinary high-tier runs.
func abyssEconomyAnomaly(depth int, bonus int64) bool {
	if bonus < 0 {
		return true
	}
	depth = max(1, depth)
	return bonus > max(int64(10_000_000), int64(depth)*2_000_000)
}

func (m *abyssOpsMetrics) observeFloor(depth int, escrowBefore int64, result abyssFloorResult, out map[string]any) {
	bonus, _ := out["bonus"].(int64)
	if abyssEconomyAnomaly(depth, bonus) || result.RewardXP < 0 || result.RewardXP > max(1_000_000, depth*100_000) {
		m.rewardAnomalies.Add(1)
		m.anomalies.Add(1)
		log.Printf("abyss reward anomaly: depth=%d bonus=%d xp=%d", depth, bonus, result.RewardXP)
	}
	for i := 1; i < len(result.Timeline); i++ {
		before, after := result.Timeline[i-1], result.Timeline[i]
		if before.EnemyMax != after.EnemyMax || before.EnemyHP <= after.EnemyHP {
			continue
		}
		burst := before.EnemyHP - after.EnemyHP
		if burst > max(1_000_000, before.EnemyMax*10) {
			m.damageAnomalies.Add(1)
			m.anomalies.Add(1)
			log.Printf("abyss damage anomaly: depth=%d burst=%d enemy_max=%d", depth, burst, before.EnemyMax)
			break
		}
	}
	if escrow, ok := out["escrow"].(int64); ok && result.Victory {
		growth := escrow - escrowBefore
		ceiling := max(int64(10_000_000), bonus+escrowBefore/2)
		if growth < bonus || growth > ceiling {
			m.economyAnomalies.Add(1)
			m.anomalies.Add(1)
			log.Printf("abyss economy anomaly: depth=%d escrow_growth=%d bonus=%d", depth, growth, bonus)
		}
	}
}

func ratio(part, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(part) / float64(total)
}

func validateAbyssContentReferences() error {
	for _, id := range []string{"small_health_potion", "great_health_potion", "elixir_of_life", "abyss_emergency_revive"} {
		if _, ok := content.GetConsumableByID(id); !ok {
			return fmt.Errorf("unknown consumable %q", id)
		}
	}
	for _, id := range []string{"ABYSS_LUCKY_COIN", "ABYSS_ARCHMAGE_ROBES"} {
		if _, ok := content.GetGearByID(id); !ok {
			return fmt.Errorf("unknown gear %q", id)
		}
	}
	for _, key := range abyssTierOrder {
		tier, ok := abyssTiers[key]
		if !ok || tier.Key != key || tier.DiffMult <= 0 || tier.RewardMult <= 0 {
			return fmt.Errorf("invalid tier %q", key)
		}
	}
	return nil
}

func abyssOpsAuthorized(r *http.Request, token string) bool {
	if token == "" {
		return false
	}
	provided := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	return len(provided) == len(token) && subtle.ConstantTimeCompare([]byte(provided), []byte(token)) == 1
}

func (s *WebServer) abyssRegistryHealth(now time.Time) (active, stale, orphan int64) {
	s.liveCombats.Range(func(_, value any) bool {
		combat, ok := value.(*abyssLiveCombat)
		if !ok || combat == nil {
			orphan++
			return true
		}
		combat.mu.Lock()
		phase, deadline, finished := combat.phase, combat.deadline, combat.finishedAt
		participants := make([]string, 0, len(combat.participants))
		for uid := range combat.participants {
			participants = append(participants, uid)
		}
		combat.mu.Unlock()
		if phase != "complete" && phase != "failed" {
			active++
		}
		planningStale := phase == "planning" && !deadline.IsZero() && now.Sub(deadline) > 2*time.Minute
		finishedStale := !finished.IsZero() && now.Sub(finished) > 10*time.Minute
		if planningStale || finishedStale {
			stale++
		}
		for _, uid := range participants {
			mapped, ok := s.liveCombatByUID.Load(uid)
			if !ok || mapped != combat.id {
				orphan++
			}
		}
		return true
	})
	s.liveCombatByUID.Range(func(_, value any) bool {
		id, ok := value.(string)
		if !ok {
			orphan++
			return true
		}
		if _, exists := s.liveCombats.Load(id); !exists {
			orphan++
		}
		return true
	})
	return active, stale, orphan
}

func durationAverage(total, count int64) int64 {
	if count <= 0 {
		return 0
	}
	return time.Duration(total / count).Milliseconds()
}

func cloneAbyssOutcomes(in map[string]abyssOpsOutcome) map[string]abyssOpsOutcome {
	out := make(map[string]abyssOpsOutcome, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

// handleAbyssOps is intentionally unavailable until ABYSS_OPS_TOKEN is set.
// The normal player session is necessary but insufficient to access operational
// health, rollout, latency, or outcome telemetry.
func (s *WebServer) handleAbyssOps(w http.ResponseWriter, r *http.Request, _ string) {
	if !abyssOpsAuthorized(r, s.abyssFeatures.opsToken) {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	active, stale, orphan := s.abyssRegistryHealth(time.Now())
	auto, manual := s.abyssOps.automaticActions.Load(), s.abyssOps.manualActions.Load()
	requestCount := s.abyssOps.requestCount.Load()
	planningCount := s.abyssOps.planningCount.Load()
	resolutionCount := s.abyssOps.resolutionCount.Load()
	s.abyssOps.mu.Lock()
	depthBands := cloneAbyssOutcomes(s.abyssOps.depthBands)
	builds := cloneAbyssOutcomes(s.abyssOps.builds)
	s.abyssOps.mu.Unlock()
	writeJSON(w, map[string]any{
		"ok":       true,
		"registry": map[string]any{"active": active, "stale": stale, "orphan": orphan},
		"actions": map[string]any{
			"automatic": auto, "manual": manual, "automatic_rate": ratio(auto, auto+manual),
		},
		"latency_ms": map[string]any{
			"request_avg":    durationAverage(s.abyssOps.requestNanos.Load(), requestCount),
			"request_max":    time.Duration(s.abyssOps.requestMaxNanos.Load()).Milliseconds(),
			"planning_avg":   durationAverage(s.abyssOps.planningNanos.Load(), planningCount),
			"resolution_avg": durationAverage(s.abyssOps.resolutionNanos.Load(), resolutionCount),
		},
		"sessions": map[string]any{
			"started": s.abyssOps.sessionsStarted.Load(), "completed": s.abyssOps.sessionsCompleted.Load(),
			"sse_connections": s.abyssOps.sseConnections.Load(),
		},
		"outcomes_by_depth": depthBands,
		"outcomes_by_build": builds,
		"anomalies": map[string]any{
			"total": s.abyssOps.anomalies.Load(), "reward": s.abyssOps.rewardAnomalies.Load(),
			"damage": s.abyssOps.damageAnomalies.Load(), "economy": s.abyssOps.economyAnomalies.Load(),
		},
		"features": map[string]any{
			"live_actions": s.abyssFeatures.liveActions, "social": s.abyssFeatures.social,
			"rollout_percent": s.abyssFeatures.rollout,
		},
	})
}
