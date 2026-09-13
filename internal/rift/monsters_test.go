package rift

import (
	"fmt"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestCatalogEncountersCoverEveryTemplate(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	seen := map[string]bool{}
	for seed := 0; seed < 4000; seed++ {
		run := NewRunWithCatalog(fmt.Sprint(seed), Build{HP: 300}, time.Unix(1, 0), catalog)
		for _, room := range run.EncounterPlan {
			for _, actor := range room {
				seen[actor.ArtKey] = true
				if actor.HP <= 0 || actor.Damage <= 0 || actor.Speed <= 0 {
					t.Fatalf("invalid actor: %+v", actor)
				}
			}
		}
	}
	for _, mob := range catalog {
		if !seen["monster:"+mob.Name] {
			t.Errorf("catalog mob never enters Brawl: %s", mob.Name)
		}
	}
}

func TestNewCatalogEntryNeedsNoBrawlRegistration(t *testing.T) {
	mob := content.Mob{Name: "Future Sentinel", Type: content.MobElite, Element: content.ElementFire, Stats: content.Stats{HP: 300, STR: 40, DEF: 22, SPD: 13}}
	run := NewRunWithCatalog("future", Build{HP: 300}, time.Unix(1, 0), []content.Mob{mob})
	if run.Enemies[0].Name != mob.Name || run.Enemies[0].ArtKey != "monster:"+mob.Name {
		t.Fatal("new catalog identity not used")
	}
	first := run.Enemies[0]
	mob.Stats.STR *= 4
	mob.Stats.SPD *= 4
	mob.Stats.DEF *= 4
	updated := AdaptMonster(mob)
	if updated.Damage <= first.Damage || updated.Speed <= first.Speed || updated.Armor <= first.Armor {
		t.Fatal("catalog stat edits did not flow into tuning")
	}
	run.Enemies[0].HP = 1
	if run.EncounterPlan[0][0].HP == 1 {
		t.Fatal("active actors alias encounter plan")
	}
}
