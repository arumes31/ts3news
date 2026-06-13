package i18n

import (
	"embed"
	"strings"
	"testing"
)

// testFS is a minimal embedded FS for testing.
// We use the real LocaleFS since it's embedded in the package.
var _ = embed.FS{}

func TestInitWithLocale(t *testing.T) {
	err := InitWithLocale(LocaleEnUS)
	if err != nil {
		t.Fatalf("InitWithLocale(en_US) failed: %v", err)
	}
	if CurrentLocale() != LocaleEnUS {
		t.Errorf("CurrentLocale() = %q, want %q", CurrentLocale(), LocaleEnUS)
	}
}

func TestT_BasicLookup(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	got := T("bot.poke.free")
	want := "Free: "
	if got != want {
		t.Errorf("T(%q) = %q, want %q", "bot.poke.free", got, want)
	}
}

func TestT_WithArgs(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	got := T("bot.combat.defeated", "Goblin", "Hero")
	want := "☠️ Goblin defeated by Hero!"
	if got != want {
		t.Errorf("T(%q) = %q, want %q", "bot.combat.defeated", got, want)
	}
}

func TestT_MultiplePositionalArgs(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	got := T("bot.combat.mob_hit", "Dragon", "Knight", 42)
	want := "💢 Dragon hits Knight for 42!"
	if got != want {
		t.Errorf("T(%q) = %q, want %q", "bot.combat.mob_hit", got, want)
	}
}

func TestT_MissingKey(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	got := T("nonexistent.key")
	if got != "nonexistent.key" {
		t.Errorf("T(missing key) = %q, want %q", got, "nonexistent.key")
	}
}

func TestT_BBCode(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	got := T("bot.pm.greeting", "Hey gamer!")
	want := "[b]Hey gamer![/b]"
	if got != want {
		t.Errorf("T(%q) = %q, want %q", "bot.pm.greeting", got, want)
	}
}

func TestT_FloatArg(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	got := T("bot.flavour.int_bonus", 1.125)
	want := "INT bonus x1.125"
	if got != want {
		t.Errorf("T(%q) = %q, want %q", "bot.flavour.int_bonus", got, want)
	}
}

func TestT_PercentLiteral(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	got := T("bot.flavour.no_game_penalty")
	if !strings.Contains(got, "-50%") {
		t.Errorf("T(%q) = %q, should contain -50%%", "bot.flavour.no_game_penalty", got)
	}
}

func TestN_EnglishSingular(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	got := N("bot.combat.reinforcements", 1)
	if !strings.Contains(got, "1 reinforcement has arrived") {
		t.Errorf("N(singular) = %q, should contain singular form", got)
	}
}

func TestN_EnglishPlural(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	got := N("bot.combat.reinforcements", 5)
	if !strings.Contains(got, "5 reinforcements have arrived") {
		t.Errorf("N(plural) = %q, should contain plural form", got)
	}
}

func TestN_MissingKey(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	got := N("nonexistent.plural", 5)
	if got != "nonexistent.plural" {
		t.Errorf("N(missing key) = %q, want %q", got, "nonexistent.plural")
	}
}

func TestP_RandomPoolEntry(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	got := P("mob.prefix")
	if got == "" {
		t.Error("P(mob.prefix) returned empty string")
	}
	pool := Pool("mob.prefix")
	found := false
	for _, entry := range pool {
		if got == entry {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("P(mob.prefix) = %q, not in pool %v", got, pool)
	}
}

func TestP_MissingPool(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	got := P("nonexistent.pool")
	if got != "" {
		t.Errorf("P(missing pool) = %q, want empty", got)
	}
}

func TestPool_FullSlice(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	pool := Pool("mob.prefix")
	if len(pool) != 10 {
		t.Errorf("Pool(mob.prefix) has %d entries, want 10", len(pool))
	}
	if pool[0] != "Ancient" {
		t.Errorf("Pool(artifact.prefix)[0] = %q, want %q", pool[0], "Ancient")
	}
}

func TestPool_GamerSuffixes(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	pool := Pool("gamer_suffix")
	if len(pool) != 8 {
		t.Errorf("Pool(gamer_suffix) has %d entries, want 8", len(pool))
	}
}

func TestR_RarityNames(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		rarity int
		want   string
	}{
		{0, "Common"},
		{1, "Uncommon"},
		{2, "Rare"},
		{3, "Epic"},
		{4, "Legendary"},
		{5, "Mythic"},
		{6, "Divine"},
	}
	for _, tt := range tests {
		got := R(tt.rarity)
		if got != tt.want {
			t.Errorf("R(%d) = %q, want %q", tt.rarity, got, tt.want)
		}
	}
}

func TestR_RarityOutOfRange(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	got := R(-1)
	if got != "Common" {
		t.Errorf("R(-1) = %q, want %q", got, "Common")
	}
	got = R(99)
	if got != "Common" {
		t.Errorf("R(99) = %q, want %q", got, "Common")
	}
}

func TestFormatGold(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	got := FormatGold(500)
	if !strings.Contains(got, "500") || !strings.Contains(got, "g") {
		t.Errorf("FormatGold(500) = %q", got)
	}
	got = FormatGold(1500)
	if !strings.Contains(got, "1.5") || !strings.Contains(got, "k") {
		t.Errorf("FormatGold(1500) = %q", got)
	}
	got = FormatGold(1_500_000)
	if !strings.Contains(got, "1.5") || !strings.Contains(got, "M") {
		t.Errorf("FormatGold(1500000) = %q", got)
	}
}

func TestFormatLarge(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	got := FormatLarge(500)
	if got != "500" {
		t.Errorf("FormatLarge(500) = %q, want %q", got, "500")
	}
	got = FormatLarge(1500)
	if !strings.Contains(got, "1.5k") {
		t.Errorf("FormatLarge(1500) = %q, should contain 1.5k", got)
	}
}

func TestFormatInt(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	got := FormatInt(1234567)
	if !strings.Contains(got, "1,234,567") {
		t.Errorf("FormatInt(1234567) = %q, should contain 1,234,567", got)
	}
}

func TestValidateFormatStrings(t *testing.T) {
	// Good: explicit positional
	good := map[string]string{
		"test.good": "Hello %[1]s",
	}
	if err := validateFormatStrings(good); err != nil {
		t.Errorf("validateFormatStrings(good) = %v, want nil", err)
	}

	// Bad: non-positional
	bad := map[string]string{
		"test.bad": "Hello %s",
	}
	if err := validateFormatStrings(bad); err == nil {
		t.Error("validateFormatStrings(bad) = nil, want error")
	}

	// OK: literal %%
	ok := map[string]string{
		"test.percent": "100%% done",
	}
	if err := validateFormatStrings(ok); err != nil {
		t.Errorf("validateFormatStrings(percent) = %v, want nil", err)
	}
}

func TestParseLocaleID(t *testing.T) {
	id, err := ParseLocaleID("en_US")
	if err != nil || id != LocaleEnUS {
		t.Errorf("ParseLocaleID(en_US) = %q, %v", id, err)
	}
	id, err = ParseLocaleID("de_DE")
	if err != nil || id != LocaleDeDE {
		t.Errorf("ParseLocaleID(de_DE) = %q, %v", id, err)
	}
	_, err = ParseLocaleID("xx_XX")
	if err == nil {
		t.Error("ParseLocaleID(xx_XX) should return error")
	}
}

func TestSetLocale(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	if err := SetLocale(LocaleDeDE); err != nil {
		// de_DE doesn't have a YAML file yet, so it may fail
		// This is expected — SetLocale only works for loaded locales
		t.Logf("SetLocale(de_DE) = %v (expected — no de_DE.yaml yet)", err)
	}
	// Setting to en_US should always work
	if err := SetLocale(LocaleEnUS); err != nil {
		t.Errorf("SetLocale(en_US) = %v", err)
	}
}

func TestPluralRules(t *testing.T) {
	tests := []struct {
		name  string
		rule  PluralRule
		count int
		want  PluralCategory
	}{
		{"English 0", ruleOneOther, 0, PluralOther},
		{"English 1", ruleOneOther, 1, PluralOne},
		{"English 2", ruleOneOther, 2, PluralOther},
		{"French 0", ruleFrenchOneOther, 0, PluralOne},
		{"French 1", ruleFrenchOneOther, 1, PluralOne},
		{"French 2", ruleFrenchOneOther, 2, PluralOther},
		{"Polish 1", rulePolish, 1, PluralOne},
		{"Polish 2", rulePolish, 2, PluralFew},
		{"Polish 5", rulePolish, 5, PluralMany},
		{"Polish 22", rulePolish, 22, PluralFew},
		{"Polish 25", rulePolish, 25, PluralMany},
		{"Russian 1", ruleRussian, 1, PluralOne},
		{"Russian 2", ruleRussian, 2, PluralFew},
		{"Russian 5", ruleRussian, 5, PluralMany},
		{"Russian 11", ruleRussian, 11, PluralMany},
		{"Russian 21", ruleRussian, 21, PluralOne},
		{"Czech 0", ruleCzech, 0, PluralZero},
		{"Czech 1", ruleCzech, 1, PluralOne},
		{"Czech 2", ruleCzech, 2, PluralFew},
		{"Czech 5", ruleCzech, 5, PluralOther},
		{"Arabic 0", ruleArabic, 0, PluralZero},
		{"Arabic 1", ruleArabic, 1, PluralOne},
		{"Arabic 2", ruleArabic, 2, PluralTwo},
		{"Arabic 3", ruleArabic, 3, PluralFew},
		{"Arabic 100", ruleArabic, 100, PluralOther},
		{"Other 1", ruleOther, 1, PluralOther},
		{"Other 5", ruleOther, 5, PluralOther},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rule(tt.count)
			if got != tt.want {
				t.Errorf("rule(%d) = %q, want %q", tt.count, got, tt.want)
			}
		})
	}
}

func TestGreetingPool(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	pool := Pool("greeting")
	if len(pool) != 100 {
		t.Errorf("Pool(greeting) has %d entries, want 100", len(pool))
	}
}

func TestLevelTierPool(t *testing.T) {
	if err := InitWithLocale(LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	pool := Pool("level.tier")
	if len(pool) != 100 {
		t.Errorf("Pool(level.tier) has %d entries, want 100", len(pool))
	}
	if pool[0] != "Drifter" {
		t.Errorf("Pool(level.tier)[0] = %q, want %q", pool[0], "Drifter")
	}
}

func TestUninitializedGlobal(t *testing.T) {
	// Save and reset global
	saved := global
	global = nil
	defer func() { global = saved }()

	// T should return key when not initialized
	if got := T("any.key"); got != "any.key" {
		t.Errorf("T() uninitialized = %q, want %q", got, "any.key")
	}
	// N should return key when not initialized
	if got := N("any.key", 5); got != "any.key" {
		t.Errorf("N() uninitialized = %q, want %q", got, "any.key")
	}
	// P should return empty when not initialized
	if got := P("any.pool"); got != "" {
		t.Errorf("P() uninitialized = %q, want empty", got)
	}
	// CurrentLocale should return default when not initialized
	if got := CurrentLocale(); got != defaultLocale {
		t.Errorf("CurrentLocale() uninitialized = %q, want %q", got, defaultLocale)
	}
}
