package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestPracticeEnemyUsesEntireLiveCatalogWithoutRewards(t *testing.T) {
	catalog := append(content.AbyssMobCatalog(), content.Mob{Name: "Future practice monster", Type: content.MobElite})
	for _, mob := range catalog {
		t.Run(mob.Name, func(t *testing.T) {
			r, err := NewPracticeRun("free", testRun().Build, "skills", time.Unix(100, 0))
			if err != nil {
				t.Fatal(err)
			}
			r.Projectiles = []Projectile{{Enemy: true, OwnerID: "old"}}
			r.Marked = "old"
			if err = r.SpawnPracticeEnemy(mob.Name, catalog); err != nil {
				t.Fatal(err)
			}
			want := AdaptMonster(mob)
			want.ID = "practice-enemy"
			want.X = 560
			want.Y = 410
			if len(r.Enemies) != 1 || r.Enemies[0] != want || len(r.Projectiles) != 0 || r.Marked != "" {
				t.Fatal("wrong roster adapter or stale combat state")
			}
			raw, _ := json.Marshal(r)
			var saved Run
			if err = json.Unmarshal(raw, &saved); err != nil {
				t.Fatal(err)
			}
			if saved.Enemies[0] != want {
				t.Fatal("selected enemy lost on save")
			}
			r.hurtEnemy(0, want.MaxHP*100, "fire")
			r.practiceTick()
			if r.Enemies[0].HP != 0 || r.Status != "fighting" || r.Gold != 0 || len(r.Drops) != 0 || len(r.MonsterRecords) != 0 || r.Stats.Kills != 0 {
				t.Fatal("practice kill gained rewards or completed lane")
			}
			if err = r.ClearPracticeEnemies(); err != nil {
				t.Fatal(err)
			}
			if len(r.Enemies) != 0 {
				t.Fatal("enemies remain")
			}
		})
	}
}

func TestPracticeEnemyInvalidActionsPreserveSnapshot(t *testing.T) {
	for _, mode := range []string{"campaign", "combo", "skills"} {
		r := testRun()
		if mode != "campaign" {
			r, _ = NewPracticeRun("test", r.Build, mode, time.Now())
		}
		before, _ := json.Marshal(r)
		if err := r.SpawnPracticeEnemy("missing", content.AbyssMobCatalog()); err == nil {
			t.Fatal("invalid spawn accepted")
		}
		if mode != "skills" {
			if err := r.ClearPracticeEnemies(); err == nil {
				t.Fatal("clear outside free practice accepted")
			}
		}
		after, _ := json.Marshal(r)
		if string(before) != string(after) {
			t.Fatal("invalid action mutated snapshot")
		}
	}
}
