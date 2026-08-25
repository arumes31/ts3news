package bot

import (
	"strings"
	"testing"
	"time"
)

func TestAbyssWatcherPressureVisibility(t *testing.T) {
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		run  abyssRun
	}{
		{name: "no run", run: abyssRun{}},
		{name: "threshold", run: abyssRun{Active: true, LastActionAt: now}},
		{name: "downed", run: abyssRun{Active: true, Depth: 8, Downed: true, LastActionAt: now}},
		{name: "missing timestamp", run: abyssRun{Active: true, Depth: 8}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := abyssWatcherPressure(tt.run, now); got.Active {
				t.Fatalf("abyssWatcherPressure(%+v).Active = true", tt.run)
			}
		})
	}
}

func TestAbyssWatcherPressureProgress(t *testing.T) {
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		last      time.Time
		percent   int
		remaining int64
		countdown string
		due       bool
	}{
		{name: "future timestamp", last: now.Add(time.Minute), percent: 0, remaining: 960, countdown: "16:00"},
		{name: "five minutes", last: now.Add(-5 * time.Minute), percent: 33, remaining: 600, countdown: "10:00"},
		{name: "threshold", last: now.Add(-15 * time.Minute), percent: 100, remaining: 0, countdown: "00:00", due: true},
		{name: "overdue", last: now.Add(-30 * time.Minute), percent: 100, remaining: 0, countdown: "00:00", due: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := abyssRun{Active: true, Depth: 8, LastActionAt: tt.last}
			got := abyssWatcherPressure(run, now)
			if !got.Active || got.Percent != tt.percent || got.RemainingSeconds != tt.remaining || got.Countdown != tt.countdown {
				t.Fatalf("abyssWatcherPressure() = %+v, want percent=%d remaining=%d countdown=%q", got, tt.percent, tt.remaining, tt.countdown)
			}
			if gotDue := abyssWatcherAmbushDue(run, now); gotDue != tt.due {
				t.Errorf("abyssWatcherAmbushDue() = %v, want %v", gotDue, tt.due)
			}
		})
	}
}

func TestAbyssWatcherPressureUIContract(t *testing.T) {
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_watcher_pressure.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"abyssWatcherPressure", "watcherPressureFill", "AMBUSH ARMED", "status.textContent", "role=\"progressbar\""} {
		if !strings.Contains(string(page), token) {
			t.Errorf("Watcher pressure UI is missing %q", token)
		}
	}
	for _, token := range []string{".ab-watcher-pressure", ".ab-watcher-pressure.is-armed", "prefers-reduced-motion"} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("Watcher pressure styles are missing %q", token)
		}
	}
}
