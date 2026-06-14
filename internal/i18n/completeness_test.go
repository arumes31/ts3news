package i18n

import (
	"fmt"
	"sort"
	"testing"
)

// loadLocaleForTest parses a single embedded locale file into a *Locale.
func loadLocaleForTest(t *testing.T, id LocaleID) *Locale {
	t.Helper()
	data, err := LocaleFS.ReadFile(fmt.Sprintf("locales/%s.yaml", id))
	if err != nil {
		t.Fatalf("locale %q is missing its YAML file: %v", id, err)
	}
	loc, err := parseLocale(data, id)
	if err != nil {
		t.Fatalf("locale %q failed to parse: %v", id, err)
	}
	return loc
}

// TestEveryLocaleHasEveryTranslation guarantees that every message key and
// every content pool defined in the canonical en_US locale also exists in all
// other supported locales. en_US is the source of truth; any key added there
// must be translated everywhere, otherwise that locale silently falls back to
// English. This test fails loudly with the exact missing keys per locale so the
// gaps are easy to close.
func TestEveryLocaleHasEveryTranslation(t *testing.T) {
	en := loadLocaleForTest(t, defaultLocale)

	if len(en.messages) == 0 {
		t.Fatalf("canonical locale %q has no messages — embed broken?", defaultLocale)
	}

	for _, id := range AllLocales {
		if id == defaultLocale {
			continue
		}
		id := id
		t.Run(string(id), func(t *testing.T) {
			loc := loadLocaleForTest(t, id)

			// 1. Every en_US message key must be present.
			var missingMsgs []string
			for key := range en.messages {
				if _, ok := loc.messages[key]; !ok {
					missingMsgs = append(missingMsgs, key)
				}
			}

			// 2. Every en_US content pool must exist with the same entry count
			//    (a short pool means missing translated lines).
			var poolProblems []string
			for pk, enEntries := range en.pools {
				got, ok := loc.pools[pk]
				if !ok {
					poolProblems = append(poolProblems, fmt.Sprintf("%s (missing pool, want %d entries)", pk, len(enEntries)))
					continue
				}
				if len(got) != len(enEntries) {
					poolProblems = append(poolProblems, fmt.Sprintf("%s (has %d entries, want %d)", pk, len(got), len(enEntries)))
				}
			}

			sort.Strings(missingMsgs)
			sort.Strings(poolProblems)

			if len(missingMsgs) > 0 {
				t.Errorf("%s is missing %d translation key(s):\n  %v", id, len(missingMsgs), missingMsgs)
			}
			if len(poolProblems) > 0 {
				t.Errorf("%s has %d content pool problem(s):\n  %v", id, len(poolProblems), poolProblems)
			}
		})
	}
}
