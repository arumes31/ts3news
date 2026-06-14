package i18n

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// placeholderRe matches explicit-index format verbs like %[1]s or %[2]d.
var placeholderRe = regexp.MustCompile(`%\[(\d+)\]([a-zA-Z])`)

// placeholders maps each referenced argument index to its format verb, ignoring
// escaped %% literals.
func placeholders(s string) map[string]string {
	clean := strings.ReplaceAll(s, "%%", "")
	out := map[string]string{}
	for _, m := range placeholderRe.FindAllStringSubmatch(clean, -1) {
		out[m[1]] = m[2]
	}
	return out
}

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

			// 3. A translation must not reference a format argument the source
			//    doesn't provide: an extra or wrong-verb index yields a runtime
			//    %!verb(BADINDEX)/type error. Omitting an optional placeholder is
			//    allowed, so only the translated side is checked against en_US.
			var placeholderProblems []string
			for key, enMsg := range en.messages {
				locMsg, ok := loc.messages[key]
				if !ok {
					continue
				}
				enP := placeholders(enMsg)
				for idx, verb := range placeholders(locMsg) {
					switch enVerb, ok := enP[idx]; {
					case !ok:
						placeholderProblems = append(placeholderProblems, fmt.Sprintf("%s references %%[%s]%s absent from source", key, idx, verb))
					case enVerb != verb:
						placeholderProblems = append(placeholderProblems, fmt.Sprintf("%s uses %%[%s]%s but source has %%[%s]%s", key, idx, verb, idx, enVerb))
					}
				}
			}

			sort.Strings(missingMsgs)
			sort.Strings(poolProblems)
			sort.Strings(placeholderProblems)

			if len(missingMsgs) > 0 {
				t.Errorf("%s is missing %d translation key(s):\n  %v", id, len(missingMsgs), missingMsgs)
			}
			if len(poolProblems) > 0 {
				t.Errorf("%s has %d content pool problem(s):\n  %v", id, len(poolProblems), poolProblems)
			}
			if len(placeholderProblems) > 0 {
				t.Errorf("%s has %d placeholder contract problem(s):\n  %v", id, len(placeholderProblems), placeholderProblems)
			}
		})
	}
}
