package bot

import (
	"sort"
	"time"
	"ts3news/internal/content"
	"ts3news/internal/rift"
)

// riftMobCatalog is a live union of the same sources used by Abyss encounters.
// Keep no copied roster or generated manifest between the two game modes.
func riftMobCatalog(now time.Time) []content.Mob {
	catalog := content.AbyssMobCatalog()
	seen := map[string]bool{}
	for _, mob := range catalog {
		seen[mob.Name] = true
	}
	add := func(mob content.Mob) {
		if !seen[mob.Name] {
			catalog = append(catalog, mob)
			seen[mob.Name] = true
		}
	}
	names := append([]string{}, abyssBossRoster...)
	names = append(names, abyssWeeklyBossNames()...)
	names = append(names, abyssBossNameAtDepth(100))
	for _, lore := range abyssBossLoreCatalog {
		names = append(names, lore.Boss)
	}
	for _, name := range names {
		add(abyssBossMob(name, 1, 1, abyssDailyBossAffinity(now).Element, 1, 1, 500))
	}
	for _, def := range abyssSecretBosses {
		for _, mob := range abyssSecretBossEncounter(def, 1, 1) {
			add(mob)
		}
	}
	sort.Slice(catalog, func(i, j int) bool { return catalog[i].Name < catalog[j].Name })
	return catalog
}

func riftBestiary(now time.Time) []rift.Actor {
	mobs := riftMobCatalog(now)
	out := make([]rift.Actor, 0, len(mobs))
	for _, mob := range mobs {
		out = append(out, rift.AdaptMonster(mob))
	}
	return out
}
