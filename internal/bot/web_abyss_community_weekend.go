package bot

import (
	"fmt"
	"time"
)

const (
	abyssCommunityWeekendGoal     int64   = 100_000_000
	abyssCommunityWeekendBonusPct         = 10
	abyssCommunityWeekendMult     float64 = 1.10
)

type abyssCommunityWeekendState struct {
	Week      string `json:"week"`
	Banked    int64  `json:"banked"`
	Target    int64  `json:"target"`
	Remaining int64  `json:"remaining"`
	Percent   int    `json:"percent"`
	Unlocked  bool   `json:"unlocked"`
	Active    bool   `json:"active"`
	BonusPct  int    `json:"bonus_pct"`
}

func abyssCommunityWeekendWindow(at time.Time) (time.Time, time.Time, string) {
	at = at.UTC()
	weekdayOffset := (int(at.Weekday()) + 6) % 7
	monday := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -weekdayOffset)
	year, week := monday.ISOWeek()
	return monday, monday.AddDate(0, 0, 7), formatAbyssISOWeek(year, week)
}

func formatAbyssISOWeek(year, week int) string {
	return fmt.Sprintf("%04d-W%02d", year, week)
}

func abyssCommunityWeekendStateFrom(at time.Time, banked int64) abyssCommunityWeekendState {
	_, _, week := abyssCommunityWeekendWindow(at)
	banked = max(int64(0), banked)
	remaining := max(int64(0), abyssCommunityWeekendGoal-banked)
	unlocked := banked >= abyssCommunityWeekendGoal
	percent := 100
	if !unlocked {
		percent = int(banked * 100 / abyssCommunityWeekendGoal)
	}
	weekend := at.UTC().Weekday() == time.Saturday || at.UTC().Weekday() == time.Sunday
	return abyssCommunityWeekendState{
		Week: week, Banked: banked, Target: abyssCommunityWeekendGoal,
		Remaining: remaining, Percent: percent, Unlocked: unlocked,
		Active: weekend && unlocked, BonusPct: abyssCommunityWeekendBonusPct,
	}
}

func (b *Bot) abyssCommunityWeekendState(at time.Time) abyssCommunityWeekendState {
	start, end, _ := abyssCommunityWeekendWindow(at)
	var banked int64
	_ = b.DB.QueryRow(
		`SELECT COALESCE(SUM(gold_banked),0) FROM abyss_runs
		 WHERE victory=TRUE AND created_at >= $1 AND created_at < $2`,
		start, end,
	).Scan(&banked)
	return abyssCommunityWeekendStateFrom(at, banked)
}

func (b *Bot) abyssCommunityWeekendRewardMult(at time.Time) float64 {
	if b.abyssCommunityWeekendState(at).Active {
		return abyssCommunityWeekendMult
	}
	return 1
}
