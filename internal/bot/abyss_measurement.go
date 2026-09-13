package bot

import (
	"runtime/debug"
	"time"
)

// Cohorts describe the producing code and economy, never inferred from a name.
type abyssMeasurementCohort struct {
	Schema        int    `json:"schema,omitempty"`
	EconomyEpoch  string `json:"economy_epoch,omitempty"`
	BuildRevision string `json:"build_revision,omitempty"`
}

// ResolutionNS is measured server resolution wall time (includes its I/O and
// scheduling). ActionWindowNS is countdown elapsed time, only an estimate of
// engagement. Neither is a denominator for player-active gold per hour.
type abyssCombatTiming struct {
	ResolutionNS   int64 `json:"resolution_ns"`
	ActionWindowNS int64 `json:"estimated_action_window_ns"`
	MeasuredFloors int   `json:"measured_floors"`
	UntimedFloors  int   `json:"untimed_floors"`
}

func abyssBuildRevision() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}
	return ""
}

func (c *abyssLiveCombat) stopResolutionLocked(now time.Time) {
	if !c.resolutionStarted.IsZero() {
		c.timing.ResolutionNS += max(int64(0), now.Sub(c.resolutionStarted).Nanoseconds())
		c.resolutionStarted = time.Time{}
	}
}
func (c *abyssLiveCombat) measurement() abyssCombatTiming {
	if c == nil {
		return abyssCombatTiming{UntimedFloors: 1}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	timing := c.timing
	if !c.resolutionStarted.IsZero() {
		timing.ResolutionNS += max(int64(0), time.Since(c.resolutionStarted).Nanoseconds())
	}
	timing.MeasuredFloors = 1
	return timing
}
func abyssEconomyLabel(recordEpoch, currentEpoch string) string {
	if recordEpoch == "" || currentEpoch == "" || recordEpoch == "unknown" || currentEpoch == "unknown" {
		return "Economy unknown"
	}
	if recordEpoch == currentEpoch {
		return "Current economy"
	}
	return "Prior economy"
}
