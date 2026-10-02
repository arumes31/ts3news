package rift

import (
	"encoding/json"
	"testing"
)

func TestSkillDamageRecordsMitigationOverkillAndPersistence(t *testing.T) {
	r := testRun()
	r.Enemies = []Actor{{ID: "armored", Kind: "goblin", ArtKey: "known", HP: 100, MaxHP: 100, Armor: .5}, {ID: "weak", Kind: "goblin", HP: 3, MaxHP: 3}}
	skill := Skill{ID: "test-fire", Kind: "fire"}
	r.skillHit(0, 20, skill, 0, "")
	r.skillHit(1, 200, skill, 0, "")
	r.skillHit(1, 200, skill, 0, "")
	if got := r.Stats.SkillDamage[skill.ID]; got != 13 {
		t.Fatalf("confirmed skill damage = %v, want 13", got)
	}
	r.hurtEnemy(0, 10, "hit")
	if r.Stats.SkillDamage[skill.ID] != 13 || r.Stats.DamageDealt <= 13 {
		t.Fatal("basic hit changed attributed skill damage")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Stats.SkillDamage[skill.ID] != 13 {
		t.Fatal("damage attribution lost on save")
	}
}

func TestSavedProjectileCreditsItsSkillDamage(t *testing.T) {
	r := testRun()
	r.Status = "cleared"
	r.Enemies = []Actor{{ID: "target", Kind: "goblin", HP: 100, MaxHP: 100, X: r.Player.X + 50, Y: r.Player.Y}}
	r.Projectiles = []Projectile{{X: r.Player.X + 50, Y: r.Player.Y, Power: 17, Life: 2, Skill: Skill{ID: "bolt", Kind: "fire"}}}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	restored.tick(Input{}, 0)
	if restored.Stats.SkillDamage["bolt"] != 17 {
		t.Fatalf("saved projectile damage = %v", restored.Stats.SkillDamage)
	}
}
