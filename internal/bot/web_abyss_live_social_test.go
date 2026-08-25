package bot

import (
	"strings"
	"testing"
	"time"

	"ts3news/internal/content"
)

func TestNormalizeAbyssSocialPreferences(t *testing.T) {
	pref := normalizeAbyssSocialPreferences(abyssSocialPreferences{Role: "wizard", Pace: "rush", Difficulty: "impossible", AllowRisky: true})
	if pref.Role != "flex" || pref.Pace != "standard" || pref.Difficulty != "any" || !pref.AllowRisky {
		t.Fatalf("unexpected normalized preferences: %#v", pref)
	}
}

func TestAbyssPartyLootRuleRoundTrip(t *testing.T) {
	for _, rule := range []string{"owner", "round_robin", "need_before_greed"} {
		if got := abyssPartyLootRuleFromID(abyssPartyLootRuleID(rule)); got != rule {
			t.Errorf("round trip %q = %q", rule, got)
		}
	}
	if got := normalizeAbyssPartyLootRule("steal_all"); got != "owner" {
		t.Fatalf("unsafe unknown rule normalized to %q", got)
	}
}

func TestMentorScaledStatsCapsPowerAtOneHundredTwentyPercent(t *testing.T) {
	host := content.Stats{HP: 100, STR: 50, DEF: 40, SPD: 30, INT: 20}
	mentor := content.Stats{HP: 1000, STR: 500, DEF: 400, SPD: 300, INT: 200}
	got := mentorScaledStats(mentor, host)
	if got.HP != 120 || got.STR != 60 || got.DEF != 48 || got.SPD != 36 || got.INT != 24 {
		t.Fatalf("mentor cap mismatch: %#v", got)
	}
}

func TestAbyssReviveVoteRequiresMajority(t *testing.T) {
	c := &abyssLiveCombat{
		participants: map[string]bool{"a": true, "b": true, "c": true},
		phase:        "complete", result: map[string]any{"downed": true},
	}
	if c.reviveApproved() {
		t.Fatal("party revive approved without votes")
	}
	if err := c.voteRevive("a", true); err != nil {
		t.Fatal(err)
	}
	if c.reviveApproved() {
		t.Fatal("one of three votes formed a majority")
	}
	if err := c.voteRevive("b", true); err != nil {
		t.Fatal(err)
	}
	if !c.reviveApproved() {
		t.Fatal("two of three votes did not form a majority")
	}
}

func TestAbyssReviveVoteCannotBePrecast(t *testing.T) {
	c := &abyssLiveCombat{participants: map[string]bool{"a": true}, phase: "planning"}
	if err := c.voteRevive("a", true); err == nil || !strings.Contains(err.Error(), "after a party defeat") {
		t.Fatalf("precast revive vote error = %v", err)
	}
}

func TestAbyssSocialSnapshotReturnsCallerPreferencesAndVotes(t *testing.T) {
	c := &abyssLiveCombat{
		participants: map[string]bool{"a": true}, phase: "planning",
		allies: []abyssLiveCombatantView{{ID: "ally:a", Name: "Ada", HP: 10}},
		social: abyssLiveSocialState{
			preferences: map[string]abyssSocialPreferences{"a": {Role: "support", Pace: "deliberate", Difficulty: "hell", AllowRisky: true}},
			tacticVotes: map[string]string{"a": "defensive"}, lastSeen: map[string]time.Time{"a": time.Now()},
		},
	}
	c.mu.Lock()
	snapshot := c.socialSnapshotLocked("a")
	c.mu.Unlock()
	if snapshot.PreferredRole != "support" || snapshot.PreferredPace != "deliberate" || snapshot.PreferredDifficulty != "hell" || !snapshot.AllowRisky || snapshot.TacticVote != "defensive" {
		t.Fatalf("caller social preferences = %#v", snapshot)
	}
}

func TestAbyssTargetPingRejectsDefeatedEnemy(t *testing.T) {
	c := &abyssLiveCombat{
		participants: map[string]bool{"a": true},
		allies:       []abyssLiveCombatantView{{ID: "ally:a", Name: "Ada", HP: 10}},
		enemies:      []abyssLiveCombatantView{{ID: "enemy:1", Name: "Gone", HP: 0}},
	}
	if err := c.addSocialSignal("a", "target", "enemy:1", ""); err == nil {
		t.Fatal("defeated enemy accepted as a target ping")
	}
}

func TestAnonymousReplayCodeContainsNoPlayerIdentityOrLogs(t *testing.T) {
	snapshot := abyssLiveSnapshot{
		Round:      7,
		Allies:     []abyssLiveCombatantView{{ID: "ally:secret-uid", Name: "Private Player"}},
		Enemies:    []abyssLiveCombatantView{{Name: "The Watcher"}},
		RecentLogs: []string{"Private Player used a secret skill"},
	}
	code, err := encodeAbyssReplayCode(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeAbyssReplayCode(code)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Rounds != 7 || len(decoded.Enemies) != 1 || decoded.Enemies[0] != "The Watcher" {
		t.Fatalf("unexpected replay summary: %#v", decoded)
	}
	if strings.Contains(code, "Private") || strings.Contains(code, "secret-uid") {
		t.Fatalf("replay code leaked player identity: %q", code)
	}
}

func TestAnonymousReplayCodeRejectsInvalidEnemyLabels(t *testing.T) {
	for _, enemy := range []string{"", strings.Repeat("x", 81)} {
		code, err := encodeAbyssReplayCode(abyssLiveSnapshot{Round: 1, Enemies: []abyssLiveCombatantView{{Name: enemy}}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodeAbyssReplayCode(code); err == nil {
			t.Fatalf("invalid enemy label accepted: %q", enemy)
		}
	}
}
