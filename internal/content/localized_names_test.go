package content

import (
	"strings"
	"testing"

	"ts3news/internal/i18n"
)

// TestInitLocalizedResolvesNames guards against the regression where content was
// generated at package-init time (before i18n was loaded), baking raw keys like
// "content.gear.novice" into item names. After InitLocalized runs with a loaded
// locale, no name may still be a raw translation key.
func TestInitLocalizedResolvesNames(t *testing.T) {
	if err := i18n.InitWithLocale(i18n.LocaleEnUS); err != nil {
		t.Fatalf("i18n init: %v", err)
	}
	InitLocalized()

	for _, g := range allGear {
		if strings.HasPrefix(g.Name, "content.") {
			t.Errorf("gear %s has unlocalized name %q", g.ID, g.Name)
		}
	}
	for _, c := range allConsumables {
		if strings.HasPrefix(c.Name, "content.") {
			t.Errorf("consumable %s has unlocalized name %q", c.ID, c.Name)
		}
	}
}

// TestMobTypeNameEliteMinion guards the snake_case key mapping: a naive
// strings.ToLower("EliteMinion") yields "eliteminion", which has no translation
// and would leak the raw key.
func TestMobTypeNameEliteMinion(t *testing.T) {
	if err := i18n.InitWithLocale(i18n.LocaleEnUS); err != nil {
		t.Fatalf("i18n init: %v", err)
	}
	for _, mt := range []MobType{MobCommon, MobEliteMinion, MobElite, MobMiniboss, MobBoss, MobLegendary} {
		name := mobTypeName(mt)
		if strings.HasPrefix(name, "content.mob.type.") {
			t.Errorf("mob type %s resolved to raw key %q", mt, name)
		}
	}
}
