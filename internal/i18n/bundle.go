package i18n

import (
	"embed"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// LocaleFS is the embedded filesystem holding locale YAML files.
// It is populated by the //go:embed directive below.
//
//go:embed locales/*.yaml
var LocaleFS embed.FS

// loadBundle reads all locale YAML files from the embedded FS and builds a Bundle.
func loadBundle(fs embed.FS, active LocaleID) (*Bundle, error) {
	b := &Bundle{
		locales: make(map[LocaleID]*Locale),
		current: active,
	}

	// Always load en_US first (it's the fallback)
	for _, id := range AllLocales {
		filename := fmt.Sprintf("locales/%s.yaml", id)
		data, err := fs.ReadFile(filename)
		if err != nil {
			if id == defaultLocale {
				return nil, fmt.Errorf("critical — en_US locale missing: %w", err)
			}
			// Non-critical: missing locale will fall back to en_US
			log.Printf("i18n: locale %q not found, will fall back to en_US", id)
			continue
		}

		loc, err := parseLocale(data, id)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", id, err)
		}

		// Validate format strings in en_US strictly; others get warnings only
		if id == defaultLocale {
			if err := validateFormatStrings(loc.messages); err != nil {
				return nil, fmt.Errorf("validate %s: %w", id, err)
			}
		} else if warn := validateFormatStrings(loc.messages); warn != nil {
			log.Printf("i18n: warning in %s: %v", id, warn)
		}

		b.locales[id] = loc
	}

	// Ensure the active locale was loaded; fall back to en_US if not
	if _, ok := b.locales[active]; !ok {
		log.Printf("i18n: active locale %q not loaded, falling back to en_US", active)
		b.current = defaultLocale
	}

	return b, nil
}

// yamlLocale is the intermediate struct for YAML deserialization.
type yamlLocale struct {
	LocaleMeta map[string]string `yaml:"_locale"`
	// All other keys are captured via inline; we use a raw map approach.
}

// parseLocale deserializes a YAML file into a Locale.
func parseLocale(data []byte, id LocaleID) (*Locale, error) {
	// First pass: parse as generic map to handle nested structures (like _locale)
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("yaml parse: %w", err)
	}

	messages := make(map[string]string)
	// Collect pool entries with indices for sorting, then build ordered slices.
	type poolEntry struct {
		index int
		value string
	}
	poolEntries := make(map[string][]poolEntry)

	for k, v := range raw {
		if k == "_locale" {
			continue
		}
		// Convert interface{} value to string
		s, ok := v.(string)
		if !ok {
			// Skip non-string values (nested maps, lists, etc.)
			continue
		}

		// Determine if this key belongs to a content pool.
		// Pool entries have a numeric index as their last segment,
		// e.g. "pool.mob.prefix.001" → pool key "mob.prefix"
		// "pool.gamer_suffix.001" → pool key "gamer_suffix"
		// "greeting.001" → pool key "greeting"
		if poolKey, idx := poolKeyFromYAMLKey(k); poolKey != "" {
			poolEntries[poolKey] = append(poolEntries[poolKey], poolEntry{index: idx, value: s})
		} else {
			messages[k] = s
		}
	}

	// Sort each pool's entries by index and build the final []string slices
	pools := make(map[string][]string, len(poolEntries))
	for pk, entries := range poolEntries {
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].index < entries[j].index
		})
		sorted := make([]string, len(entries))
		for i, e := range entries {
			sorted[i] = e.value
		}
		pools[pk] = sorted
	}

	// Determine plural rule from locale ID
	pluralRule := pluralRuleForLocale(id)

	// Get number format
	numFmt := numberFormatForLocale(id)

	// Allow YAML overrides for number suffixes
	if v, ok := messages["number.suffix.k"]; ok {
		numFmt.SuffixK = v
		delete(messages, "number.suffix.k")
	}
	if v, ok := messages["number.suffix.m"]; ok {
		numFmt.SuffixM = v
		delete(messages, "number.suffix.m")
	}
	if v, ok := messages["number.suffix.b"]; ok {
		numFmt.SuffixB = v
		delete(messages, "number.suffix.b")
	}
	if v, ok := messages["number.gold_suffix"]; ok {
		numFmt.GoldSuffix = v
		delete(messages, "number.gold_suffix")
	}

	return &Locale{
		ID:       id,
		Plural:   pluralRule,
		messages: messages,
		pools:    pools,
		numFmt:   numFmt,
	}, nil
}

// poolIndexRe matches a numeric index suffix like ".001", ".100", etc.
var poolIndexRe = regexp.MustCompile(`\.\d{2,3}$`)

// poolKeyFromYAMLKey determines whether a YAML key represents a pool entry.
// If it does, it returns the pool key (without "pool." prefix and without the
// numeric index) and the numeric index. Otherwise it returns ("", 0).
//
// Examples:
//
//	"pool.mob.prefix.001"    → ("mob.prefix",    1)
//	"pool.gamer_suffix.001"  → ("gamer_suffix",  1)
//	"pool.level.tier.001"    → ("level.tier",    1)
//	"pool.level.epic_realm.030" → ("level.epic_realm", 30)
//	"greeting.001"           → ("greeting",      1)
//	"bot.poke.free"          → ("", 0)  — not a pool entry
func poolKeyFromYAMLKey(key string) (string, int) {
	// Check if the key ends with a numeric index like ".001"
	if !poolIndexRe.MatchString(key) {
		return "", 0
	}

	// Strip the "pool." prefix if present
	body := key
	if strings.HasPrefix(key, "pool.") {
		body = key[len("pool."):]
	}

	// Split off the last segment (the numeric index)
	lastDot := strings.LastIndex(body, ".")
	if lastDot < 0 {
		return "", 0
	}

	indexStr := body[lastDot+1:]
	body = body[:lastDot]

	index := 0
	for _, ch := range indexStr {
		index = index*10 + int(ch-'0')
	}

	return body, index
}

// nonPositionalVerb matches non-positional format verbs like %s, %d, %f, etc.
// but NOT %[1]s, %%, or literal %%.
var nonPositionalVerb = regexp.MustCompile(`%(?:\d*\.?\d*)[sdfeEgGxXouUbcqvp]`)

// validateFormatStrings ensures all messages use explicit positional arguments.
// This prevents translation bugs where word-order changes break implicit %s/%d.
func validateFormatStrings(msgs map[string]string) error {
	var errs []string
	for key, msg := range msgs {
		if nonPositionalVerb.MatchString(msg) {
			errs = append(errs, fmt.Sprintf("key %q: non-positional format verb in %q — use %%[N]s style", key, msg))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}
