package bot

import (
	"strings"
	"testing"
	"time"

	"ts3news/internal/content"
	"ts3news/internal/rift"
)

func TestRiftCatalogIdentityMatchesEverySharedSource(t *testing.T) {
	// Exercise a full week of date-dependent boss affinities, not just one day.
	for day := 0; day < 7; day++ {
		now := time.Date(2026, 9, 13+day, 12, 0, 0, 0, time.UTC)
		t.Run(now.Format("2006-01-02"), func(t *testing.T) {
			expected := map[string]content.Mob{}
			add := func(mob content.Mob) {
				if strings.TrimSpace(mob.Name) == "" {
					t.Fatal("shared source contains an unnamed monster")
				}
				// Canonical templates take precedence over named boss supplements.
				if _, exists := expected[mob.Name]; !exists {
					expected[mob.Name] = mob
				}
			}
			for _, mob := range content.AbyssMobCatalog() {
				if _, exists := expected[mob.Name]; exists {
					t.Fatalf("duplicate canonical identity %q", mob.Name)
				}
				add(mob)
			}
			addBoss := func(name string) {
				add(abyssBossMob(name, 1, 1, abyssDailyBossAffinity(now).Element, 1, 1, 500))
			}
			for _, name := range abyssBossRoster {
				addBoss(name)
			}
			for _, name := range abyssWeeklyBossNames() {
				addBoss(name)
			}
			addBoss(abyssBossNameAtDepth(100))
			for _, lore := range abyssBossLoreCatalog {
				addBoss(lore.Boss)
			}
			for _, secret := range abyssSecretBosses {
				encounter := abyssSecretBossEncounter(secret, 1, 1)
				if len(encounter) == 0 {
					t.Fatalf("secret boss %q has no encounter", secret.Name)
				}
				for _, mob := range encounter {
					add(mob)
				}
			}
			actual := riftBestiary(now)
			if len(actual) != len(expected) {
				t.Fatalf("got %d identities, shared sources have %d", len(actual), len(expected))
			}
			seen := map[string]bool{}
			for _, actor := range actual {
				source, exists := expected[actor.Name]
				if !exists {
					t.Fatalf("Brawl-only identity %q is absent from shared sources", actor.Name)
				}
				if seen[actor.Name] {
					t.Fatalf("duplicate Brawl identity %q", actor.Name)
				}
				seen[actor.Name] = true
				if actor.ArtKey != "monster:"+source.Name || actor.Tier != string(source.Type) || actor.Element != strings.ToLower(string(source.Element)) {
					t.Fatalf("identity drift for %q: art=%q tier=%q element=%q", actor.Name, actor.ArtKey, actor.Tier, actor.Element)
				}
				isBoss := source.Type == content.MobBoss || source.Type == content.MobLegendary
				if (actor.Kind == "boss") != isBoss {
					t.Fatalf("boss-pool classification drift for %q (%s)", actor.Name, source.Type)
				}
				// A one-entry source proves eligibility without relying on a lucky
				// random seed from the entire live roster.
				run := rift.NewRunWithCatalog("identity-check", rift.Build{HP: 100}, now, []content.Mob{source})
				for room, actors := range run.EncounterPlan {
					if len(actors) == 0 {
						t.Fatalf("%q has no planned actors in room %d", actor.Name, room)
					}
					for _, planned := range actors {
						if planned.Name != actor.Name || planned.ArtKey != actor.ArtKey || planned.Tier != actor.Tier || planned.Element != actor.Element {
							t.Fatalf("encounter planning changed %q identity in room %d", actor.Name, room)
						}
					}
				}
			}
			t.Logf("validated %d shared identities", len(expected))
		})
	}
}
