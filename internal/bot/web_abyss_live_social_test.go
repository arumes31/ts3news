package bot

import (
	"strings"
	"testing"

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
	c := &abyssLiveCombat{participants: map[string]bool{"a": true, "b": true, "c": true}}
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
