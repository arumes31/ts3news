package bot

import (
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestRiftRosterTracksAllAbyssCatalogs(t *testing.T) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	got := riftBestiary(now)
	seen := map[string]bool{}
	for _, mob := range got {
		if seen[mob.Name] {
			t.Fatalf("duplicate monster %s", mob.Name)
		}
		seen[mob.Name] = true
		if mob.ArtKey != "monster:"+mob.Name {
			t.Fatal("art identity drift")
		}
	}
	for _, mob := range content.AbyssMobCatalog() {
		if !seen[mob.Name] {
			t.Errorf("missing regular monster %s", mob.Name)
		}
	}
	for _, name := range abyssBossRoster {
		if !seen[name] {
			t.Errorf("missing named boss %s", name)
		}
	}
	for _, name := range abyssWeeklyBossNames() {
		if !seen[name] {
			t.Errorf("missing world boss %s", name)
		}
	}
	for _, boss := range abyssBossLoreCatalog {
		if !seen[boss.Boss] {
			t.Errorf("missing lore boss %s", boss.Boss)
		}
	}
	for _, boss := range abyssSecretBosses {
		if !seen[boss.Name] {
			t.Errorf("missing secret boss %s", boss.Name)
		}
	}
	t.Logf("Brawl roster: %d canonical monsters", len(got))
}

func TestRiftBestiaryPreservesCanonicalAbyssTier(t *testing.T) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	expected := map[string]string{}
	for _, mob := range riftMobCatalog(now) {
		expected[mob.Name] = string(mob.Type)
	}
	for _, actor := range riftBestiary(now) {
		if actor.Tier != expected[actor.Name] || actor.Tier == "" {
			t.Fatalf("tier drift for %s: %s, expected %s", actor.Name, actor.Tier, expected[actor.Name])
		}
	}
}
