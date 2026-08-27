package bot

import "testing"

func TestNormalizeAbyssTactic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "aggressive", input: "Aggressive", expected: "aggressive"},
		{name: "defensive", input: " defensive ", expected: "defensive"},
		{name: "conserve items", input: "conserve_items", expected: "conserve_items"},
		{name: "unknown falls back", input: "reckless", expected: "balanced"},
		{name: "empty falls back", input: "", expected: "balanced"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := normalizeAbyssTactic(test.input); got != test.expected {
				t.Fatalf("normalizeAbyssTactic(%q) = %q, want %q", test.input, got, test.expected)
			}
		})
	}
}

func TestAbyssLiveSnapshotIncludesSchemaVersion(t *testing.T) {
	combat := &abyssLiveCombat{
		id:           "session",
		ownerUID:     "user",
		participants: map[string]bool{"user": true},
		tactics:      map[string]string{"user": "balanced"},
		options:      map[string][]abyssLiveOption{},
		queued:       map[string]abyssLiveAction{},
	}

	snapshot := combat.snapshotFor("user")
	if snapshot.SchemaVersion != abyssLiveSnapshotSchemaVersion {
		t.Fatalf("snapshot schema version = %d, want %d", snapshot.SchemaVersion, abyssLiveSnapshotSchemaVersion)
	}
}

func TestAbyssLiveSnapshotRestoresCompleteLogHistory(t *testing.T) {
	t.Parallel()

	first := abyssLiveSnapshot{LogStart: 0, LogCursor: 2, RecentLogs: []string{"round one", "round two"}}
	second := abyssLiveSnapshot{LogStart: 2, LogCursor: 4, RecentLogs: []string{"round three", "round four"}}
	combat := &abyssLiveCombat{
		id:           "resume-session",
		ownerUID:     "user",
		participants: map[string]bool{"user": true},
		tactics:      map[string]string{"user": "balanced"},
		options:      map[string][]abyssLiveOption{},
		queued:       map[string]abyssLiveAction{},
		recentLogs:   append([]string(nil), second.RecentLogs...),
		lastLogCount: 4,
		history: []abyssLiveEvent{
			{ID: 1, Snapshots: map[string]abyssLiveSnapshot{"user": first}},
			// Multiple state versions in one round carry the same cursor. They
			// overwrite their positions rather than duplicating feed entries.
			{ID: 2, Snapshots: map[string]abyssLiveSnapshot{"user": first}},
			{ID: 3, Snapshots: map[string]abyssLiveSnapshot{"user": second}},
		},
	}

	snapshot := combat.snapshotWithLogHistory("user")
	want := []string{"round one", "round two", "round three", "round four"}
	if len(snapshot.LogHistory) != len(want) {
		t.Fatalf("log history = %q, want %q", snapshot.LogHistory, want)
	}
	for index := range want {
		if snapshot.LogHistory[index] != want[index] {
			t.Fatalf("log history[%d] = %q, want %q", index, snapshot.LogHistory[index], want[index])
		}
	}
	if snapshot.LogStart != 2 || snapshot.LogCursor != 4 {
		t.Fatalf("current log range = [%d,%d), want [2,4)", snapshot.LogStart, snapshot.LogCursor)
	}
}

func TestAbyssLiveSnapshotAdvertisesBossEnrageRound(t *testing.T) {
	t.Parallel()

	combat := &abyssLiveCombat{
		id:           "boss-session",
		ownerUID:     "user",
		participants: map[string]bool{"user": true},
		tactics:      map[string]string{"user": "balanced"},
		options:      map[string][]abyssLiveOption{},
		queued:       map[string]abyssLiveAction{},
		enemies:      []abyssLiveCombatantView{{Name: "Abyssus", Role: "boss", HP: 100, MaxHP: 100}},
	}
	if got := combat.snapshotFor("user").EnrageRound; got != 30 {
		t.Fatalf("Abyss boss enrage round = %d, want 30", got)
	}
	combat.enemies[0].Role = "bruiser"
	if got := combat.snapshotFor("user").EnrageRound; got != 0 {
		t.Fatalf("ordinary encounter enrage round = %d, want 0", got)
	}
}

func TestCombatBossEnrageThresholdMatchesAdvertisedRound(t *testing.T) {
	t.Parallel()

	if combatBossShouldEnrage(29, true) || !combatBossShouldEnrage(30, true) {
		t.Fatal("Abyss boss enrage does not begin on advertised round 30")
	}
	if combatBossShouldEnrage(7, false) || !combatBossShouldEnrage(8, false) {
		t.Fatal("standard boss enrage does not begin on round 8")
	}
}
