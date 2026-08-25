package bot

import (
	"fmt"
	"time"
)

type abyssWatcherPressureView struct {
	Active            bool
	Percent           int
	RemainingSeconds  int64
	Countdown         string
	DeadlineUnixMilli int64
	NowUnixMilli      int64
	ThresholdSeconds  int64
}

func abyssWatcherPressure(run abyssRun, now time.Time) abyssWatcherPressureView {
	if !run.Active || run.Downed || run.Depth <= 0 || run.LastActionAt.IsZero() {
		return abyssWatcherPressureView{}
	}

	deadline := run.LastActionAt.Add(abyssWatcherIdle)
	remaining := deadline.Sub(now)
	if remaining < 0 {
		remaining = 0
	}
	elapsed := now.Sub(run.LastActionAt)
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed > abyssWatcherIdle {
		elapsed = abyssWatcherIdle
	}

	remainingSeconds := int64((remaining + time.Second - 1) / time.Second)
	percent := int(elapsed * 100 / abyssWatcherIdle)
	return abyssWatcherPressureView{
		Active:            true,
		Percent:           percent,
		RemainingSeconds:  remainingSeconds,
		Countdown:         fmt.Sprintf("%02d:%02d", remainingSeconds/60, remainingSeconds%60),
		DeadlineUnixMilli: deadline.UnixMilli(),
		NowUnixMilli:      now.UnixMilli(),
		ThresholdSeconds:  int64(abyssWatcherIdle / time.Second),
	}
}

func abyssWatcherAmbushDue(run abyssRun, now time.Time) bool {
	return run.Depth > 0 && !run.LastActionAt.IsZero() && !now.Before(run.LastActionAt.Add(abyssWatcherIdle))
}
