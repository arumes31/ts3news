// Package i18n provides lightweight internationalization for ts3news.
// It loads YAML locale files embedded in the binary and provides
// translation functions T(), N() (plural), P() (pool random), and R() (rarity).
package i18n

import (
	"embed"
	"fmt"
	"log"
	"math/rand/v2"
	"regexp"
	"strings"
)

// LocaleID is a BCP 47 locale identifier, e.g. "en_US", "de_DE".
type LocaleID string

// Supported locales.
const (
	LocaleEnUS LocaleID = "en_US"
	LocaleDeDE LocaleID = "de_DE"
	LocaleEsES LocaleID = "es_ES"
	LocaleFrFR LocaleID = "fr_FR"
	LocaleItIT LocaleID = "it_IT"
	LocalePtBR LocaleID = "pt_BR"
	LocaleJaJP LocaleID = "ja_JP"
	LocaleKoKR LocaleID = "ko_KR"
	LocaleZhCN LocaleID = "zh_CN"
	LocaleZhTW LocaleID = "zh_TW"
	LocaleRuRU LocaleID = "ru_RU"
	LocalePlPL LocaleID = "pl_PL"
	LocaleTrTR LocaleID = "tr_TR"
	LocaleNlNL LocaleID = "nl_NL"
	LocaleSvSE LocaleID = "sv_SE"
	LocaleCsCZ LocaleID = "cs_CZ"
	LocaleArSA LocaleID = "ar_SA"
	LocaleThTH LocaleID = "th_TH"
	LocaleViVN LocaleID = "vi_VN"
	LocaleHiIN LocaleID = "hi_IN"
)

// defaultLocale is the fallback for missing translations.
const defaultLocale LocaleID = LocaleEnUS

// AllLocales is the ordered list of all supported locale IDs.
var AllLocales = []LocaleID{
	LocaleEnUS, LocaleDeDE, LocaleEsES, LocaleFrFR, LocaleItIT,
	LocalePtBR, LocaleJaJP, LocaleKoKR, LocaleZhCN, LocaleZhTW,
	LocaleRuRU, LocalePlPL, LocaleTrTR, LocaleNlNL, LocaleSvSE,
	LocaleCsCZ, LocaleArSA, LocaleThTH, LocaleViVN, LocaleHiIN,
}

// global is the package-level singleton, set by Init.
var global *Bundle

// Init loads all embedded locale files and sets the active locale.
// Must be called once at program startup before any T()/N()/P() calls.
func Init(fs embed.FS, locale LocaleID) error {
	b, err := loadBundle(fs, locale)
	if err != nil {
		return fmt.Errorf("i18n.Init: %w", err)
	}
	global = b
	return nil
}

// InitWithLocale is a convenience that uses the package-level embedded FS.
func InitWithLocale(locale LocaleID) error {
	return Init(LocaleFS, locale)
}

// SetLocale changes the active locale at runtime.
func SetLocale(id LocaleID) error {
	if global == nil {
		return fmt.Errorf("i18n: not initialized")
	}
	if _, ok := global.locales[id]; !ok {
		return fmt.Errorf("i18n: unknown locale %q", id)
	}
	global.current = id
	return nil
}

// CurrentLocale returns the active locale ID.
func CurrentLocale() LocaleID {
	if global == nil {
		return defaultLocale
	}
	return global.current
}

// T translates a message key with the given arguments.
// It uses the current locale, falling back to en_US, then the key itself.
//
// Format verbs in translation strings MUST use explicit positional indices:
//
//	"%[1]s defeated %[2]s" — not "%s defeated %s"
func T(key string, args ...any) string {
	if global == nil {
		return key
	}
	return global.translate(key, args...)
}

// N translates a message with plural form selection.
//
//	count = 1  → looks up key + ".one"
//	count != 1 → looks up key + ".other"
//
// The count value is available as %[1]d in the format string.
func N(key string, count int, args ...any) string {
	if global == nil {
		return key
	}
	return global.translatePlural(key, count, args...)
}

// P returns a random entry from a named content pool for the current locale.
// Falls back to en_US pool if the current locale's pool is empty.
func P(pool string) string {
	if global == nil {
		return ""
	}
	return global.randomPoolEntry(pool)
}

// Pool returns the full slice of entries for a named content pool.
// Falls back to en_US if the current locale has no entries.
func Pool(pool string) []string {
	if global == nil {
		return nil
	}
	return global.getPool(pool)
}

// R returns the translated rarity name for a given rarity constant (0-6).
func R(rarity int) string {
	keys := []string{
		"rarity.common", "rarity.uncommon", "rarity.rare",
		"rarity.epic", "rarity.legendary", "rarity.mythic", "rarity.divine",
	}
	if rarity < 0 || rarity >= len(keys) {
		return T("rarity.common")
	}
	return T(keys[rarity])
}

// FormatGold formats a gold value with locale-appropriate suffixes and BBCode.
func FormatGold(v int64) string {
	if global == nil {
		return fmt.Sprintf("%d", v)
	}
	return global.formatGold(v)
}

// FormatLarge formats a large number with locale-appropriate suffixes (no BBCode).
func FormatLarge(v float64) string {
	if global == nil {
		return fmt.Sprintf("%.0f", v)
	}
	return global.formatLarge(v)
}

// FormatInt formats an integer with locale-appropriate grouping.
func FormatInt(n int64) string {
	if global == nil {
		return fmt.Sprintf("%d", n)
	}
	return global.formatInt(n)
}

// --- Bundle ---

// Bundle holds all loaded locale data and provides translation methods.
type Bundle struct {
	locales map[LocaleID]*Locale
	current LocaleID
}

// Locale holds all translations for a single locale.
type Locale struct {
	ID       LocaleID
	Plural   PluralRule
	messages map[string]string
	pools    map[string][]string // "mob.prefix" → ["Snotty", "Angry", ...]
	numFmt   NumberFormat
}

// translate looks up key in current locale → en_US fallback → key itself.
func (b *Bundle) translate(key string, args ...any) string {
	msg := b.lookup(key)
	if msg == "" {
		return key
	}
	return formatMsg(msg, args...)
}

// lookup finds the message in current locale, then en_US fallback.
func (b *Bundle) lookup(key string) string {
	// 1. Try current locale
	if loc, ok := b.locales[b.current]; ok {
		if msg, ok := loc.messages[key]; ok {
			return msg
		}
	}
	// 2. Fallback to en_US
	if b.current != defaultLocale {
		if loc, ok := b.locales[defaultLocale]; ok {
			if msg, ok := loc.messages[key]; ok {
				return msg
			}
		}
	}
	return ""
}

// translatePlural selects the correct plural form and then translates.
func (b *Bundle) translatePlural(key string, count int, args ...any) string {
	loc := b.locales[b.current]
	if loc == nil {
		loc = b.locales[defaultLocale]
	}
	if loc == nil {
		return key
	}

	form := loc.Plural.Form(count)

	// Try specific plural form, then "other"
	pluralKey := key + "." + string(form)
	msg := b.lookup(pluralKey)
	if msg == "" && form != PluralOther {
		msg = b.lookup(key + ".other")
	}
	if msg == "" {
		return key
	}

	// Prepend count to args so %[1]d is always the count
	allArgs := make([]any, 0, 1+len(args))
	allArgs = append(allArgs, count)
	allArgs = append(allArgs, args...)
	return formatMsg(msg, allArgs...)
}

// randomPoolEntry picks a random entry from a named pool.
func (b *Bundle) randomPoolEntry(pool string) string {
	entries := b.getPool(pool)
	if len(entries) == 0 {
		return ""
	}
	return entries[rand.IntN(len(entries))]
}

// getPool returns the pool entries for the current locale, falling back to en_US.
func (b *Bundle) getPool(pool string) []string {
	if loc, ok := b.locales[b.current]; ok {
		if entries, ok := loc.pools[pool]; ok && len(entries) > 0 {
			return entries
		}
	}
	// Fallback to en_US
	if loc, ok := b.locales[defaultLocale]; ok {
		if entries, ok := loc.pools[pool]; ok {
			return entries
		}
	}
	return nil
}

// positionalFloatRe matches %[N].Pf or %[N]W.Pf patterns in format strings.
// Go's fmt does NOT support %[1].3f — positional indices cannot be combined
// with width/precision. We work around this by pre-formatting such arguments
// and replacing the verb with %[N]s.
//
// Match groups: 1=index, 2=width (optional), 3=.precision (optional), 4=verb
var positionalFloatRe = regexp.MustCompile(`%\[(\d+)\](\d*)?(\.\d+)?([fFeEgG])`)

// formatMsg formats a message pattern with the given arguments.
// It works around Go's fmt limitation where %[N].Pf is invalid by
// pre-formatting float arguments that use precision/width with positional
// indices, then substituting %[N]s into the pattern.
func formatMsg(pattern string, args ...any) string {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("i18n: format error for pattern %q with %d args: %v", pattern, len(args), r)
		}
	}()

	// Check if the pattern has any positional float verbs that need pre-formatting
	if !positionalFloatRe.MatchString(pattern) {
		return fmt.Sprintf(pattern, args...)
	}

	// Find all positional float verbs and pre-format them
	result := pattern
	matches := positionalFloatRe.FindAllStringSubmatchIndex(pattern, -1)

	// Process matches in reverse to preserve indices
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		argIdx := 0
		for _, c := range pattern[m[2]:m[3]] {
			argIdx = argIdx*10 + int(c-'0')
		}
		widthStr := ""
		if m[4] >= 0 && m[5] >= 0 {
			widthStr = pattern[m[4]:m[5]]
		}
		precStr := ""
		if m[6] >= 0 && m[7] >= 0 {
			precStr = pattern[m[6]:m[7]]
		}
		verb := pattern[m[8]:m[9]]

		// argIdx is 1-based
		if argIdx < 1 || argIdx > len(args) {
			continue
		}

		// Build a non-positional format verb: %W.Pf
		fmtVerb := "%" + widthStr + precStr + verb

		// Format the argument
		formatted := fmt.Sprintf(fmtVerb, args[argIdx-1])

		// Replace the positional verb with %[N]s and swap the arg to the formatted string
		result = result[:m[0]] + fmt.Sprintf("%%[%d]s", argIdx) + result[m[1]:]

		// Replace the original arg with the pre-formatted string
		args[argIdx-1] = formatted
	}

	return fmt.Sprintf(result, args...)
}

// numberFormat returns the NumberFormat for the current locale.
func (b *Bundle) numberFormat() NumberFormat {
	if loc, ok := b.locales[b.current]; ok {
		return loc.numFmt
	}
	if loc, ok := b.locales[defaultLocale]; ok {
		return loc.numFmt
	}
	return defaultNumberFormat()
}

// formatGold formats a gold value with locale suffixes and BBCode.
func (b *Bundle) formatGold(v int64) string {
	nf := b.numberFormat()
	if v < 1000 {
		return fmt.Sprintf("[b]%s[/b][color=#9e9e9e]%s[/color]", applyDigitTransform(fmt.Sprintf("%d", v), nf.DigitTransform), nf.GoldSuffix)
	}
	if v < 1_000_000 {
		return fmt.Sprintf("[b]%s[/b][color=#9e9e9e]%s[/color]",
			applyDigitTransform(formatFloat(float64(v)/1000.0, 1, nf), nf.DigitTransform), nf.SuffixK)
	}
	if v < 1_000_000_000 {
		return fmt.Sprintf("[b]%s[/b][color=#9e9e9e]%s[/color]",
			applyDigitTransform(formatFloat(float64(v)/1_000_000.0, 1, nf), nf.DigitTransform), nf.SuffixM)
	}
	return fmt.Sprintf("[b]%s[/b][color=#9e9e9e]%s[/color]",
		applyDigitTransform(formatFloat(float64(v)/1_000_000_000.0, 1, nf), nf.DigitTransform), nf.SuffixB)
}

// formatLarge formats a large number with locale suffixes (no BBCode).
func (b *Bundle) formatLarge(v float64) string {
	nf := b.numberFormat()
	if v < 1000 {
		return applyDigitTransform(formatFloat(v, 0, nf), nf.DigitTransform)
	}
	if v < 1_000_000 {
		return applyDigitTransform(formatFloat(v/1000.0, 1, nf), nf.DigitTransform) + nf.SuffixK
	}
	if v < 1_000_000_000 {
		return applyDigitTransform(formatFloat(v/1_000_000.0, 2, nf), nf.DigitTransform) + nf.SuffixM
	}
	return applyDigitTransform(formatFloat(v/1_000_000_000.0, 2, nf), nf.DigitTransform) + nf.SuffixB
}

// formatInt formats an integer with locale-appropriate grouping.
func (b *Bundle) formatInt(n int64) string {
	nf := b.numberFormat()
	s := fmt.Sprintf("%d", n)
	if nf.GroupSize <= 0 || nf.GroupSep == "" {
		return applyDigitTransform(s, nf.DigitTransform)
	}

	negative := false
	if n < 0 {
		negative = true
		s = s[1:]
	}

	// Group from right
	parts := []string{}
	for len(s) > nf.GroupSize {
		parts = append([]string{s[len(s)-nf.GroupSize:]}, parts...)
		s = s[:len(s)-nf.GroupSize]
	}
	if len(s) > 0 {
		parts = append([]string{s}, parts...)
	}

	result := strings.Join(parts, nf.GroupSep)
	if negative {
		result = "-" + result
	}
	return applyDigitTransform(result, nf.DigitTransform)
}

// formatFloat formats a float with locale-appropriate decimal separator.
func formatFloat(v float64, decimals int, nf NumberFormat) string {
	fmtStr := fmt.Sprintf("%%.%df", decimals)
	s := fmt.Sprintf(fmtStr, v)
	if nf.DecimalSep != "." {
		s = strings.Replace(s, ".", nf.DecimalSep, 1)
	}
	return s
}

// ParseLocaleID converts a string to a LocaleID, returning an error if unsupported.
func ParseLocaleID(s string) (LocaleID, error) {
	id := LocaleID(s)
	for _, supported := range AllLocales {
		if id == supported {
			return id, nil
		}
	}
	return "", fmt.Errorf("i18n: unsupported locale %q", s)
}
