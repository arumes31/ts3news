package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAbyssWorldBossWeekendUsesUTCBoundaries(t *testing.T) {
	tests := []struct {
		at   time.Time
		want int
	}{
		{time.Date(2026, time.August, 28, 23, 59, 59, 0, time.UTC), 1},
		{time.Date(2026, time.August, 29, 0, 0, 0, 0, time.UTC), 2},
		{time.Date(2026, time.August, 30, 23, 59, 59, 0, time.UTC), 2},
		{time.Date(2026, time.August, 31, 0, 0, 0, 0, time.UTC), 1},
	}
	for _, test := range tests {
		if got := abyssWorldBossStrikeMultiplier(test.at); got != test.want {
			t.Errorf("multiplier at %s = %d, want %d", test.at, got, test.want)
		}
	}
	localSaturday := time.Date(2026, time.August, 29, 1, 0, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	if abyssWorldBossWeekend(localSaturday) {
		t.Fatal("Friday 23:00 UTC was treated as weekend from local wall time")
	}
}

func TestAbyssWorldBossWeekendDoublesDamageAndDropTogether(t *testing.T) {
	drop := abyssWeeklyBossDrop{Material: "core", Amount: 3, Weight: 100}
	weekday := time.Date(2026, time.August, 28, 12, 0, 0, 0, time.UTC)
	damage, reward := applyAbyssWorldBossWeekendReward(weekday, 700, drop)
	if damage != 700 || reward.Amount != 3 {
		t.Fatalf("weekday reward = %d / %+v", damage, reward)
	}
	weekend := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	damage, reward = applyAbyssWorldBossWeekendReward(weekend, 700, drop)
	if damage != 1400 || reward.Amount != 6 {
		t.Fatalf("weekend reward = %d / %+v", damage, reward)
	}
}

func TestAbyssWorldBossWeekendUIContract(t *testing.T) {
	root := abyssAAARepositoryRoot(t)
	path := filepath.Join(root, "internal", "bot", "webassets", "abyss_social.html")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"WEEKEND RAID SURGE", ".WeeklyBoss.Multiplier", "Saturday–Sunday UTC", "2× damage &amp; loot"} {
		if !strings.Contains(string(raw), token) {
			t.Errorf("%s is missing %q", filepath.Base(path), token)
		}
	}
}
