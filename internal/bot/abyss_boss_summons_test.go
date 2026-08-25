package bot

import (
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssBossSummonChoreography(t *testing.T) {
	bosses := []string{
		"Gorgoroth the Firelord",
		"Malakor the Voidweaver",
		"Azazoth the Slumbering Eye",
		"Abyssus, Heart of the Void",
	}
	telegraphs := make(map[string]bool, len(bosses))
	arrivals := make(map[string]bool, len(bosses))
	prefixes := make(map[string]bool, len(bosses))

	for _, boss := range bosses {
		choreography := abyssBossSummonFor(boss)
		for _, required := range []string{boss, "summoning ritual", "ULTIMATE"} {
			if !strings.Contains(choreography.Telegraph, required) {
				t.Errorf("%s telegraph %q does not contain %q", boss, choreography.Telegraph, required)
			}
		}
		if choreography.Arrival == "" || choreography.AddPrefix == "" {
			t.Errorf("%s has incomplete choreography: %+v", boss, choreography)
		}
		telegraphs[choreography.Telegraph] = true
		arrivals[choreography.Arrival] = true
		prefixes[choreography.AddPrefix] = true

		users := []activeUser{{u: &UserInCombat{UID: "delver", CurrentHP: 100}}}
		mobs := []*content.Mob{{Name: boss, Type: content.MobBoss, Stats: content.Stats{HP: 100}}}
		plans := planLiveEnemyIntents(1, users, mobs, []string{choreography.Telegraph})
		if got := plans[0].Intent; got.Kind != "interruptible" || !strings.Contains(got.Ability, "Ultimate") {
			t.Errorf("%s live intent = %+v, want interruptible Ultimate window", boss, got)
		}
	}

	if len(telegraphs) != len(bosses) || len(arrivals) != len(bosses) || len(prefixes) != len(bosses) {
		t.Fatalf("named choreography is not distinct: telegraphs=%d arrivals=%d prefixes=%d", len(telegraphs), len(arrivals), len(prefixes))
	}
}

func TestAbyssBossSummonChoreographyFallback(t *testing.T) {
	const boss = "Unknown Horror"
	choreography := abyssBossSummonFor(boss)
	if choreography.AddPrefix != "Summoned" {
		t.Fatalf("fallback prefix = %q, want Summoned", choreography.AddPrefix)
	}
	for _, required := range []string{boss, "summoning ritual", "ULTIMATE"} {
		if !strings.Contains(choreography.Telegraph, required) {
			t.Errorf("fallback telegraph %q does not contain %q", choreography.Telegraph, required)
		}
	}
	if !strings.Contains(choreography.Arrival, boss) || !strings.Contains(choreography.Arrival, "reinforcements arrive") {
		t.Errorf("fallback arrival = %q", choreography.Arrival)
	}
}
