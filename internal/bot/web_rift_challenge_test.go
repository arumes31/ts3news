package bot

import (
	"testing"
	"time"
)

func TestRiftChallengeDeterministic(t *testing.T) {
	date := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	ch := riftChallenge(date)
	if ch.Key == "" || ch.Label == "" || ch.Criteria == "" {
		t.Fatalf("expected non-empty challenge fields, got %+v", ch)
	}

	for _, key := range abyssDailyMods {
		c := riftChallengeFor(key)
		if c.Key != key {
			t.Errorf("expected key %s, got %s", key, c.Key)
		}
		if c.Label == "" || c.Criteria == "" {
			t.Errorf("expected non-empty label and criteria for %s", key)
		}
		if len(c.Difficulties) == 0 {
			t.Errorf("expected at least one compatible difficulty for %s", key)
		}
	}
}
