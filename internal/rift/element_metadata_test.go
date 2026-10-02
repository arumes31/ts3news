package rift

import (
	"encoding/json"
	"testing"
	"ts3news/internal/content"
)

func TestProjectileKeepsCanonicalElementAcrossSave(t *testing.T) {
	r := testRun()
	r.Build.Skills = []Skill{{ID: "element-test", Name: "Ice-looking fire", Kind: "ice", Element: content.ElementFire, Power: 1, Cost: 12, Cooldown: 2}}
	r.Player.Mana, r.Player.Cooldown = 100, 0
	r.cast("element-test")
	if len(r.Projectiles) != 1 || r.Projectiles[0].Skill.Element != content.ElementFire || r.Projectiles[0].Kind != "ice" {
		t.Fatal("cast lost canonical element")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Projectiles[0].Skill.Element != content.ElementFire {
		t.Fatal("saved projectile lost element")
	}
	var legacy Skill
	if err = json.Unmarshal([]byte(`{"id":"legacy","kind":"fire"}`), &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.Element != "" {
		t.Fatal("legacy element inferred from artwork")
	}
}
