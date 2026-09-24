//go:build e2e

package bot

import (
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"time"
	"ts3news/internal/rift"
)

var riftVisualSeedPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// Fixed content time also fixes daily boss affinity. Only authored content and
// the selected build/seed/mission/tier can change this paused simulation snapshot.
func riftVisualFixture(query url.Values, build rift.Build) (*rift.Run, error) {
	seed := query.Get("seed")
	if seed == "" {
		seed = "ruins-v1"
	}
	if !riftVisualSeedPattern.MatchString(seed) {
		return nil, errors.New("seed must contain 1-64 letters, digits, hyphens or underscores")
	}
	level, room := 1, 0
	for _, option := range []struct {
		name     string
		target   *int
		min, max int
	}{{"level", &level, 1, rift.LevelCount}, {"room", &room, 0, len(rift.Rooms) - 1}} {
		if raw := query.Get(option.name); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < option.min || value > option.max {
				return nil, errors.New("invalid visual fixture " + option.name)
			}
			*option.target = value
		}
	}
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	run := rift.NewRunAtLevel("visual-"+seed, build, now, riftMobCatalog(now), level)
	for run.Room < room {
		run.Status = "cleared"
		run.NextRoom()
	}
	run.Epoch = "fixture"
	run.SetPaused(true, now)
	return run, nil
}
