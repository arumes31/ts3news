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
