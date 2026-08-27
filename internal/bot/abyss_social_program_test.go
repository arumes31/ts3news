package bot

import (
	"strings"
	"testing"
	"time"

	"ts3news/internal/content"
)

func TestAbyssHelperDepthRewardScalesAndCaps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		depth int
		want  int
	}{{-1, 5}, {0, 5}, {10, 6}, {50, 10}, {150, 20}, {999, 20}}
	for _, test := range tests {
		if got := abyssHelperDepthReward(test.depth); got != test.want {
			t.Fatalf("abyssHelperDepthReward(%d) = %d, want %d", test.depth, got, test.want)
		}
	}
}

func TestAbyssSocialGoalAndPresenceBounds(t *testing.T) {
	t.Parallel()
	if got := abyssSocialGoalPercent(-1); got != 0 {
		t.Fatalf("negative progress = %d, want 0", got)
	}
	if got := abyssSocialGoalPercent(abyssDailyServerGoal * 2); got != 100 {
		t.Fatalf("over-cap progress = %d, want 100", got)
	}
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	if !abyssSocialOnline(now.Add(-10*time.Minute), now) {
		t.Fatal("ten-minute boundary should be online")
	}
	if abyssSocialOnline(now.Add(-10*time.Minute-time.Second), now) {
		t.Fatal("presence older than ten minutes should be offline")
	}
}

func TestScaleAbyssCheerStatsOnlyScalesCombatResources(t *testing.T) {
	t.Parallel()
	stats := content.Stats{HP: 100, STR: 80, DEF: 60, SPD: 40, INT: 20, MNA: 200, LCK: 13}
	got := scaleAbyssCheerStats(stats)
	if got.HP != 105 || got.STR != 84 || got.DEF != 63 || got.SPD != 42 || got.INT != 21 || got.MNA != 210 {
		t.Fatalf("unexpected cheer stats: %+v", got)
	}
	if got.LCK != stats.LCK {
		t.Fatalf("cheer changed non-combat luck: %d", got.LCK)
	}
}

func TestAbyssDuelResolveIsSideEffectFreeAndDeterministic(t *testing.T) {
	t.Parallel()
	first := content.Stats{HP: 100, STR: 30, DEF: 20, SPD: 50}
	second := content.Stats{HP: 100, STR: 20, DEF: 10, SPD: 40}
	winner, logs := abyssDuelResolve("First", first, "Second", second)
	if winner != 0 || len(logs) < 3 {
		t.Fatalf("winner=%d logs=%d", winner, len(logs))
	}
	if first.HP != 100 || second.HP != 100 {
		t.Fatal("duel mutated input stats")
	}
}

func TestAbyssSocialProgramUIWiresServerAuthoritativeActions(t *testing.T) {
	t.Parallel()
	partial, err := webAssets.ReadFile("webassets/abyss_social.html")
	if err != nil {
		t.Fatal(err)
	}
	page := string(partial)
	for _, token := range []string{
		"/api/abyss/social/friend", "/api/abyss/social/friend/cheer",
		"/api/abyss/social/mentor", "/api/abyss/social/referral",
		"/api/abyss/social/trade", "/api/abyss/social/shoutbox",
		"/api/abyss/social/floor_message", "/api/abyss/social/kudos",
		"/api/abyss/social/guild", "/api/abyss/social/tournament",
		"/api/abyss/social/duel", "/api/abyss/social/raid", "/api/abyss/social/rescue",
		"/api/abyss/coop/leave",
	} {
		if !strings.Contains(page, token) {
			t.Errorf("Fellowship UI missing %q", token)
		}
	}
	styles, err := webAssets.ReadFile("webassets/abyss_social.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{".ab-program-card", ".ab-trade-grid", ".ab-board-columns"} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("Fellowship styles missing %q", token)
		}
	}
}
